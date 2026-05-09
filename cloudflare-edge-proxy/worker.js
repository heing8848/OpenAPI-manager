/**
 * AlfredAPI Cloudflare Edge Proxy Worker V2
 *
 * V2 preserves the original request bytes until the central gateway confirms
 * the request should execute through edge proxy. The server now prepares the
 * final upstream URL, headers, and JSON body so edge execution stays in parity
 * with the normal relay path.
 */

const WORKER_RELAY_PATH_HEADER_V2 = "X-Alfred-Relay-Path";

export default {
	async fetch(request, env, ctx) {
		const ALFRED_API_HOST = env.ALFRED_API_HOST;
		const WORKER_ADMIN_TOKEN = env.WORKER_ADMIN_TOKEN;

		if (!ALFRED_API_HOST || !WORKER_ADMIN_TOKEN) {
			return new Response("Missing Edge Proxy configuration", { status: 500 });
		}

		if (request.method === "OPTIONS") {
			return new Response(null, {
				headers: corsHeadersV2(),
			});
		}

		const requestURL = new URL(request.url);
		const originalBodyBuffer = request.method === "POST" ? await request.clone().arrayBuffer() : null;
		if (!shouldHandleEdgeRelayPathV2(requestURL)) {
			return forwardOriginalRequestToOriginV2(request, ALFRED_API_HOST, originalBodyBuffer);
		}
		if (request.method !== "POST") {
			return forwardOriginalRequestToOriginV2(request, ALFRED_API_HOST, originalBodyBuffer);
		}

		const authHeader = request.headers.get("Authorization");
		if (!authHeader) {
			return jsonResponseV2({ error: { message: "Missing Authorization header" } }, 401);
		}

		const startTime = Date.now();
		const relayPath = `${requestURL.pathname}${requestURL.search}`;
		const prepareRes = await fetch(`${ALFRED_API_HOST}/api/worker/prepare_v2`, {
			method: "POST",
			headers: buildPrepareHeadersV2(request, authHeader, relayPath),
			body: originalBodyBuffer ? originalBodyBuffer.slice(0) : undefined,
		});

		if (prepareRes.status === 403) {
			const prepareErrorText = await prepareRes.text();
			if (shouldFallbackToOriginV2(prepareErrorText)) {
				return forwardOriginalRequestToOriginV2(request, ALFRED_API_HOST, originalBodyBuffer);
			}
			return new Response(prepareErrorText, {
				status: prepareRes.status,
				headers: withCorsHeadersV2(prepareRes.headers),
			});
		}
		if (!prepareRes.ok) {
			return mirrorGatewayErrorV2(prepareRes);
		}

		const prepareInfo = await prepareRes.json();
		let upstreamRes;
		try {
			upstreamRes = await fetch(prepareInfo.target_url, {
				method: prepareInfo.method || "POST",
				headers: new Headers(prepareInfo.headers || {}),
				body: prepareInfo.body,
			});
		} catch (error) {
			const errorMsg = `Upstream Network Error: ${error.toString()}`;
			ctx.waitUntil(reportUsageV2(
				ALFRED_API_HOST,
				WORKER_ADMIN_TOKEN,
				prepareInfo,
				0,
				0,
				Date.now() - startTime,
				false,
				true,
				errorMsg,
				"upstream_error",
				"network_error",
				502,
				""
			));
			return jsonResponseV2({
				error: {
					message: errorMsg,
					type: "upstream_error",
					param: "502",
					code: "network_error",
				},
			}, 502);
		}

		if (!prepareInfo.is_stream || !upstreamRes.ok) {
			return handleBufferedUpstreamResponseV2(
				upstreamRes,
				prepareInfo,
				ALFRED_API_HOST,
				WORKER_ADMIN_TOKEN,
				startTime,
				ctx
			);
		}

		return handleStreamingUpstreamResponseV2(
			upstreamRes,
			prepareInfo,
			ALFRED_API_HOST,
			WORKER_ADMIN_TOKEN,
			startTime,
			ctx
		);
	},
};

function corsHeadersV2() {
	return {
		"Access-Control-Allow-Origin": "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	};
}

function withCorsHeadersV2(headers) {
	const newHeaders = new Headers(headers || {});
	newHeaders.set("Access-Control-Allow-Origin", "*");
	return newHeaders;
}

function buildOriginURLV2(host, requestURL) {
	const originURL = new URL(host);
	originURL.pathname = requestURL.pathname;
	originURL.search = requestURL.search;
	return originURL.toString();
}

function buildPrepareHeadersV2(request, authHeader, relayPath) {
	const headers = new Headers();
	headers.set("Authorization", authHeader);
	headers.set(WORKER_RELAY_PATH_HEADER_V2, relayPath);

	const contentType = request.headers.get("Content-Type");
	if (contentType) {
		headers.set("Content-Type", contentType);
	}
	const accept = request.headers.get("Accept");
	if (accept) {
		headers.set("Accept", accept);
	}
	return headers;
}

function shouldHandleEdgeRelayPathV2(requestURL) {
	return requestURL.pathname.startsWith("/v1/");
}

function buildForwardHeadersV2(request) {
	const headers = new Headers(request.headers);
	headers.delete("Host");
	return headers;
}

async function forwardOriginalRequestToOriginV2(request, host, originalBodyBuffer) {
	const requestURL = new URL(request.url);
	const init = {
		method: request.method,
		headers: buildForwardHeadersV2(request),
	};
	if (request.method !== "GET" && request.method !== "HEAD" && originalBodyBuffer != null) {
		init.body = originalBodyBuffer.slice(0);
	}
	return fetch(buildOriginURLV2(host, requestURL), init);
}

async function mirrorGatewayErrorV2(response) {
	const body = await response.arrayBuffer();
	return new Response(body, {
		status: response.status,
		headers: withCorsHeadersV2(response.headers),
	});
}

function jsonResponseV2(payload, status, extraHeaders = {}) {
	const headers = withCorsHeadersV2(extraHeaders);
	headers.set("Content-Type", "application/json");
	return new Response(JSON.stringify(payload), { status, headers });
}

function appendRequestIdV2(message, requestId) {
	if (!requestId || String(message).includes(requestId)) {
		return message;
	}
	if (!message) {
		return `(request id: ${requestId})`;
	}
	return `${message} (request id: ${requestId})`;
}

function shouldFallbackToOriginV2(responseText) {
	try {
		const parsed = JSON.parse(responseText);
		const errorCode = parsed?.error?.code;
		return errorCode === "edge_proxy_disabled" || errorCode === "worker_prepare_mode_not_supported";
	} catch {
		return false;
	}
}

function normalizeUpstreamErrorPayloadV2(status, rawText, requestId) {
	let parsed;
	try {
		parsed = rawText ? JSON.parse(rawText) : null;
	} catch {
		parsed = null;
	}

	if (parsed && typeof parsed === "object") {
		const errorObject = parsed.error && typeof parsed.error === "object" ? parsed.error : null;
		const errorType = errorObject?.type || "upstream_error";
		const errorCode = errorObject?.code || "bad_response_status_code";
		const message = appendRequestIdV2(errorObject?.message || rawText || `bad response status code ${status}`, requestId);

		if (errorObject) {
			errorObject.message = message;
		} else {
			parsed.error = {
				message,
				type: errorType,
				param: String(status),
				code: errorCode,
			};
		}

		return {
			bodyText: JSON.stringify(parsed),
			errorMessage: message,
			errorType,
			errorCode,
		};
	}

	const message = appendRequestIdV2((rawText || "").trim() || `bad response status code ${status}`, requestId);
	return {
		bodyText: JSON.stringify({
			error: {
				message,
				type: "upstream_error",
				param: String(status),
				code: "bad_response_status_code",
			},
		}),
		errorMessage: message,
		errorType: "upstream_error",
		errorCode: "bad_response_status_code",
	};
}

function extractUsageFromBodyV2(bodyText) {
	try {
		const parsed = JSON.parse(bodyText);
		const usage = parsed?.usage;
		return {
			promptTokens: usage?.prompt_tokens || 0,
			completionTokens: usage?.completion_tokens || 0,
		};
	} catch {
		return {
			promptTokens: 0,
			completionTokens: 0,
		};
	}
}

async function handleBufferedUpstreamResponseV2(upstreamRes, prepareInfo, host, adminToken, startTime, ctx) {
	const rawText = await upstreamRes.text();
	const requestId = upstreamRes.headers.get("x-oneapi-request-id") || upstreamRes.headers.get("x-request-id") || "";
	const headers = withCorsHeadersV2(upstreamRes.headers);
	const isError = !upstreamRes.ok;

	if (!isError) {
		const usage = extractUsageFromBodyV2(rawText);
		ctx.waitUntil(reportUsageV2(
			host,
			adminToken,
			prepareInfo,
			usage.promptTokens,
			usage.completionTokens,
			Date.now() - startTime,
			false,
			false,
			"",
			"",
			"",
			upstreamRes.status,
			requestId
		));
		return new Response(rawText, {
			status: upstreamRes.status,
			headers,
		});
	}

	const normalized = normalizeUpstreamErrorPayloadV2(upstreamRes.status, rawText, requestId);
	ctx.waitUntil(reportUsageV2(
		host,
		adminToken,
		prepareInfo,
		0,
		0,
		Date.now() - startTime,
		false,
		true,
		normalized.errorMessage,
		normalized.errorType,
		normalized.errorCode,
		upstreamRes.status,
		requestId
	));
	headers.set("Content-Type", "application/json");
	return new Response(normalized.bodyText, {
		status: upstreamRes.status,
		headers,
	});
}

function approximatePromptTokensV2(bodyText) {
	try {
		const parsed = JSON.parse(bodyText);
		return JSON.stringify(parsed.messages || []).length / 2;
	} catch {
		return 0;
	}
}

function handleStreamingUpstreamResponseV2(upstreamRes, prepareInfo, host, adminToken, startTime, ctx) {
	const responseHeaders = withCorsHeadersV2(upstreamRes.headers);
	const { readable, writable } = new TransformStream();
	const reader = upstreamRes.body.getReader();
	const writer = writable.getWriter();
	const decoder = new TextDecoder("utf-8");
	const requestBodyText = prepareInfo.body || "";

	ctx.waitUntil((async () => {
		let promptTokens = 0;
		let completionTokens = 0;
		let buffer = "";
		try {
			while (true) {
				const { done, value } = await reader.read();
				if (done) {
					break;
				}
				await writer.write(value);

				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split("\n");
				buffer = lines.pop() || "";

				for (const line of lines) {
					if (!line.startsWith("data: ") || line.includes("[DONE]")) {
						continue;
					}
					try {
						const dataObj = JSON.parse(line.slice(6).trim());
						if (dataObj.usage?.total_tokens) {
							promptTokens = dataObj.usage.prompt_tokens || 0;
							completionTokens = dataObj.usage.completion_tokens || 0;
						} else if (dataObj.choices?.[0]?.delta?.content) {
							completionTokens += dataObj.choices[0].delta.content.length / 1.5;
						}
					} catch {
						// Ignore malformed SSE chunks and continue streaming.
					}
				}
			}
		} catch (error) {
			console.error("Stream pipe error", error);
		} finally {
			await writer.close();
			if (promptTokens === 0) {
				promptTokens = approximatePromptTokensV2(requestBodyText);
			}
			await reportUsageV2(
				host,
				adminToken,
				prepareInfo,
				promptTokens,
				completionTokens,
				Date.now() - startTime,
				true,
				false,
				"",
				"",
				"",
				upstreamRes.status,
				""
			);
		}
	})());

	return new Response(readable, {
		status: upstreamRes.status,
		headers: responseHeaders,
	});
}

async function reportUsageV2(
	host,
	adminToken,
	prepareInfo,
	promptTokens,
	completionTokens,
	elapsed,
	isStream,
	isError = false,
	errorMsg = "",
	errorType = "",
	errorCode = "",
	statusCode = 200,
	requestId = ""
) {
	try {
		await fetch(`${host}/api/worker/callback_v2`, {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"Authorization": `Bearer ${adminToken}`,
			},
			body: JSON.stringify({
				user_id: prepareInfo.user_id,
				token_id: prepareInfo.token_id,
				token_name: prepareInfo.token_name,
				model: prepareInfo.actual_model,
				prompt_tokens: Math.ceil(promptTokens),
				completion_tokens: Math.ceil(completionTokens),
				channel_id: prepareInfo.channel_id,
				channel_key_id: prepareInfo.channel_key_id,
				channel_key_index: prepareInfo.channel_key_index,
				elapsed_time: elapsed,
				is_stream: isStream,
				is_error: isError,
				error_msg: errorMsg,
				error_type: errorType,
				error_code: errorCode,
				status_code: statusCode,
				request_id: requestId,
			}),
		});
	} catch (error) {
		console.error("Failed to report usage backwards", error);
	}
}
