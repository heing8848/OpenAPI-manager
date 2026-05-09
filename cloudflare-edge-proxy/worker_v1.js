/**
 * AlfredAPI Cloudflare Edge Proxy Worker
 *
 * This edge worker intercepts API requests (like /v1/chat/completions),
 * authenticates/routes via the AlfredAPI central server, and then directly 
 * connects to the upstream provider (e.g., Nvidia, OpenRouter, ModelScope).
 * It streams the response directly to the client with ZERO server detour, 
 * then asynchronously reports the token usage back to AlfredAPI.
 *
 * Environment Variables required:
 * - ALFRED_API_HOST: "https://<your-alfredapi-domain>"
 * - WORKER_ADMIN_TOKEN: "<alfredapi-admin-access-token>"
 */

export default {
	async fetch(request, env, ctx) {
		const ALFRED_API_HOST = env.ALFRED_API_HOST;
		const WORKER_ADMIN_TOKEN = env.WORKER_ADMIN_TOKEN;

		if (!ALFRED_API_HOST || !WORKER_ADMIN_TOKEN) {
			return new Response("Missing Edge Proxy configuration", { status: 500 });
		}

		if (request.method === "OPTIONS") {
			// Handle CORS preflight
			return new Response(null, {
				headers: {
					"Access-Control-Allow-Origin": "*",
					"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
					"Access-Control-Allow-Headers": "Content-Type, Authorization"
				}
			});
		}

		if (request.method !== "POST") {
			// Simply proxy non-POST requests to main server if needed, or deny.
			// Currently this worker is optimized for POST generation paths.
			const fallbackHeaders = new Headers(request.headers);
			fallbackHeaders.delete("Host");
			return fetch(`${ALFRED_API_HOST}${new URL(request.url).pathname}`, {
				method: request.method,
				headers: fallbackHeaders,
				body: request.body // For non-POST, stream hasn't been touched yet
			});
		}

		const startTime = Date.now();
		const clonedReq = request.clone();
		const authHeader = request.headers.get("Authorization");

		if (!authHeader) {
			return new Response(JSON.stringify({ error: { message: "Missing Authorization header" } }), { status: 401, headers: { "Content-Type": "application/json" } });
		}

		let bodyJson;
		try {
			bodyJson = await clonedReq.json();
		} catch (e) {
			return new Response("Invalid JSON body", { status: 400 });
		}

		const requestedModel = bodyJson.model || "";
		const isStream = !!bodyJson.stream;

		// Inject stream_options so upstream tells us the usage in stream tail
		if (isStream) {
			if (!bodyJson.stream_options) {
				bodyJson.stream_options = { include_usage: true };
			} else {
				bodyJson.stream_options.include_usage = true;
			}
		}

		// Phase 1: Authentication and Route Discovery
		// This uses a lightweight call to fetch the true BaseURL and Upstream API Key without transmitting heavy data.
		const routeRes = await fetch(`${ALFRED_API_HOST}/api/worker/route`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"Authorization": authHeader
			},
			body: JSON.stringify({ model: requestedModel })
		});

		if (!routeRes.ok) {
			// e.g. 403 No Edge Proxy configured, 401 Invalid Token, 503 No Channel found
			if (routeRes.status === 403) {
				// Fallback to normal proxy if edge proxy is not enabled for this channel
				const fallbackHeaders = new Headers(request.headers);
				fallbackHeaders.delete("Host");
				
				return fetch(`${ALFRED_API_HOST}${new URL(request.url).pathname}`, {
					method: request.method,
					headers: fallbackHeaders,
					body: JSON.stringify(bodyJson) // Re-serialize since original body stream is consumed
				});
			}
			const errorData = await routeRes.text();
			return new Response(`[Edge Proxy] Central Gateway Rejected: ${errorData}`, { status: routeRes.status });
		}

		const routeInfo = await routeRes.json();
		const {
			base_url, key, channel_id, user_id,
			token_id, token_name, system_prompt, actual_model
		} = routeInfo;

		// Inject actual mapped model
		if (actual_model) {
			bodyJson.model = actual_model;
		}

		// Inject system prompt if mandated by channel setting
		if (system_prompt && bodyJson.messages && Array.isArray(bodyJson.messages) && bodyJson.messages.length > 0) {
			if (bodyJson.messages[0].role === "system") {
				bodyJson.messages[0].content = system_prompt;
			} else {
				bodyJson.messages.unshift({ role: "system", content: system_prompt });
			}
		}

		// Phase 2: Direct Execution to Upstream Provider
		let upstreamUrlPath = new URL(request.url).pathname; // e.g. /v1/chat/completions
		let cleanBaseUrl = base_url.endsWith('/') ? base_url.slice(0, -1) : base_url;
		if (cleanBaseUrl.endsWith('/v1') && upstreamUrlPath.startsWith('/v1/')) {
			cleanBaseUrl = cleanBaseUrl.slice(0, -3);
		}
		const targetUrl = cleanBaseUrl + upstreamUrlPath;

		let upstreamRes;
		try {
			upstreamRes = await fetch(targetUrl, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"Authorization": `Bearer ${key}`
				},
				body: JSON.stringify(bodyJson)
			});
		} catch (e) {
			ctx.waitUntil(reportUsage(
				ALFRED_API_HOST, WORKER_ADMIN_TOKEN,
				user_id, token_id, token_name, requestedModel,
				0, 0, channel_id, Date.now() - startTime, false, true, "Upstream Network Error: " + e.toString()
			));
			return new Response(`[Edge Proxy] Upstream Network Error: ${e.toString()}`, { 
				status: 502,
				headers: { "Access-Control-Allow-Origin": "*" }
			});
		}

		let promptTokens = 0;
		let completionTokens = 0;

		const newHeaders = new Headers(upstreamRes.headers);
		newHeaders.set("Access-Control-Allow-Origin", "*");

		// 2A. Non-Stream Handling OR Error Handling
		if (!isStream || !upstreamRes.ok) {
			let finalData = {};
			let errorMsg = "";
			let isError = !upstreamRes.ok;
			
			try {
				finalData = await upstreamRes.json();
				isError = isError || (finalData && finalData.error != null);
				if (isError) {
					errorMsg = JSON.stringify(finalData);
				}
			} catch (e) {
				isError = isError || (finalData && finalData.error != null);
				if (isError) {
					errorMsg = "Failed to parse upstream error response";
				}
			}

			if (finalData.usage) {
				promptTokens = finalData.usage.prompt_tokens || 0;
				completionTokens = finalData.usage.completion_tokens || 0;
			}

			ctx.waitUntil(reportUsage(
				ALFRED_API_HOST, WORKER_ADMIN_TOKEN,
				user_id, token_id, token_name, requestedModel,
				promptTokens, completionTokens, channel_id, Date.now() - startTime, false, isError, errorMsg
			));

			return new Response(JSON.stringify(finalData), {
				status: upstreamRes.status,
				headers: newHeaders
			});
		}

		// 2B. Streaming Handling - Zero Latency Throughput
		const { readable, writable } = new TransformStream();
		const reader = upstreamRes.body.getReader();
		const writer = writable.getWriter();
		const decoder = new TextDecoder("utf-8");

		ctx.waitUntil((async () => {
			let buffer = "";
			try {
				while (true) {
					const { done, value } = await reader.read();
					if (done) break;

					// Push immediately to the client to ensure Zero-Delay TTFT
					await writer.write(value);

					// Then decode & count silently in background
					buffer += decoder.decode(value, { stream: true });
					const lines = buffer.split('\n');
					buffer = lines.pop(); // keep remainder

					for (const line of lines) {
						if (line.startsWith("data: ") && !line.includes("[DONE]")) {
							try {
								const dataObj = JSON.parse(line.substring(6).trim());
								if (dataObj.usage && dataObj.usage.total_tokens) {
									promptTokens = dataObj.usage.prompt_tokens || 0;
									completionTokens = dataObj.usage.completion_tokens || 0;
								} else if (dataObj.choices && dataObj.choices[0]?.delta?.content) {
									// Fallback estimation
									completionTokens += (dataObj.choices[0].delta.content.length / 1.5);
								}
							} catch (e) { }
						}
					}
				}
			} catch (err) {
				console.error("Stream pipe error", err);
			} finally {
				await writer.close();

				// Fallback approximation
				if (promptTokens === 0) {
					promptTokens = JSON.stringify(bodyJson.messages || []).length / 2;
				}

				return reportUsage(
					ALFRED_API_HOST, WORKER_ADMIN_TOKEN,
					user_id, token_id, token_name, requestedModel,
					promptTokens, completionTokens, channel_id, Date.now() - startTime, true
				);
			}
		})());

		return new Response(readable, {
			status: upstreamRes.status,
			headers: newHeaders
		});
	}
};

// Async utility to push callback gracefully without keeping client waiting
async function reportUsage(host, adminToken, userId, tokenId, tokenName, model, promptT, completionT, channelId, elapsed, isStream, isError = false, errorMsg = "") {
	try {
		await fetch(`${host}/api/worker/callback`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"Authorization": `Bearer ${adminToken}`
			},
			body: JSON.stringify({
				user_id: userId,
				token_id: tokenId,
				token_name: tokenName,
				model: model,
				prompt_tokens: Math.ceil(promptT),
				completion_tokens: Math.ceil(completionT),
				channel_id: channelId,
				elapsed_time: elapsed,
				is_stream: isStream,
				is_error: isError,
				error_msg: errorMsg
			})
		});
	} catch (e) {
		console.error("Failed to report usage backwards", e);
	}
}
