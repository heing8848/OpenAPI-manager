import test from "node:test";
import assert from "node:assert/strict";

import workerModule from "./worker.js";

const env = {
	ALFRED_API_HOST: "https://api.alfred.example",
	WORKER_ADMIN_TOKEN: "admin-token",
};

function createContext() {
	return {
		waitUntil(promise) {
			this.promises.push(Promise.resolve(promise));
		},
		promises: [],
	};
}

test("fallback keeps the original request bytes untouched when prepare_v2 returns 403", async () => {
	const rawBody = '{"model":"mistral-large-latest","messages":[{"role":"user","content":"hello"}],"stream":true}';
	const capturedBodies = [];
	const originalFetch = globalThis.fetch;

	globalThis.fetch = async (input, init = {}) => {
		const url = typeof input === "string" ? input : input.url;
		if (url.endsWith("/api/worker/prepare_v2")) {
			return new Response(JSON.stringify({
				error: {
					message: "channel does not support edge proxy",
					type: "invalid_request_error",
					code: "edge_proxy_disabled",
				},
			}), {
				status: 403,
				headers: { "Content-Type": "application/json" },
			});
		}

		capturedBodies.push(await new Response(init.body).text());
		return new Response('{"ok":true}', {
			status: 200,
			headers: { "Content-Type": "application/json" },
		});
	};

	try {
		const request = new Request("https://edge.alfred.example/v1/chat/completions?foo=bar", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"Authorization": "Bearer test",
			},
			body: rawBody,
		});
		const ctx = createContext();
		const response = await workerModule.fetch(request, env, ctx);

		assert.equal(response.status, 200);
		assert.equal(capturedBodies.length, 1);
		assert.equal(capturedBodies[0], rawBody);
		await Promise.allSettled(ctx.promises);
	} finally {
		globalThis.fetch = originalFetch;
	}
});

test("non-json upstream errors are wrapped into an OpenAI-style error payload", async () => {
	const rawBody = '{"model":"mistral-large-latest","messages":[{"role":"user","content":"hello"}],"stream":false}';
	const originalFetch = globalThis.fetch;

	globalThis.fetch = async (input, init = {}) => {
		const url = typeof input === "string" ? input : input.url;
		if (url.endsWith("/api/worker/prepare_v2")) {
			return new Response(JSON.stringify({
				channel_id: 9,
				channel_key_id: 12,
				channel_key_index: 1,
				target_url: "https://api.mistral.ai/v1/chat/completions",
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					"Authorization": "Bearer upstream",
				},
				body: rawBody,
				actual_model: "mistral-large-latest",
				user_id: 7,
				token_id: 8,
				token_name: "edge token",
				is_stream: false,
			}), {
				status: 200,
				headers: { "Content-Type": "application/json" },
			});
		}
		if (url.endsWith("/api/worker/callback_v2")) {
			return new Response('{"status":"success"}', {
				status: 200,
				headers: { "Content-Type": "application/json" },
			});
		}
		return new Response("Access denied", {
			status: 403,
			headers: {
				"Content-Type": "text/plain; charset=utf-8",
				"x-oneapi-request-id": "req-123",
			},
		});
	};

	try {
		const request = new Request("https://edge.alfred.example/v1/chat/completions", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
				"Authorization": "Bearer sk-test",
			},
			body: rawBody,
		});
		const ctx = createContext();
		const response = await workerModule.fetch(request, env, ctx);
		const payload = await response.json();

		assert.equal(response.status, 403);
		assert.equal(payload.error.type, "upstream_error");
		assert.equal(payload.error.code, "bad_response_status_code");
		assert.match(payload.error.message, /Access denied/);
		assert.match(payload.error.message, /req-123/);
		await Promise.allSettled(ctx.promises);
	} finally {
		globalThis.fetch = originalFetch;
	}
});

test("non-v1 requests are transparently forwarded to origin without edge prepare", async () => {
	const capturedUrls = [];
	const originalFetch = globalThis.fetch;

	globalThis.fetch = async (input) => {
		const url = typeof input === "string" ? input : input.url;
		capturedUrls.push(url);
		return new Response('{"success":true}', {
			status: 200,
			headers: { "Content-Type": "application/json" },
		});
	};

	try {
		const request = new Request("https://edge.alfred.example/api/status", {
			method: "GET",
		});
		const ctx = createContext();
		const response = await workerModule.fetch(request, env, ctx);
		const payload = await response.json();

		assert.equal(response.status, 200);
		assert.deepEqual(payload, { success: true });
		assert.deepEqual(capturedUrls, ["https://api.alfred.example/api/status"]);
		await Promise.allSettled(ctx.promises);
	} finally {
		globalThis.fetch = originalFetch;
	}
});
