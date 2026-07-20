/**
 * rmb memory capture for OpenCode
 *
 * On session idle, fetches the latest user/assistant turn and pipes a JSON
 * payload to `rmb hook-submit --source=opencode`.
 *
 * Install: copy this file to ~/.config/opencode/plugin/rmb-hook.ts
 *
 * Env:
 *   RMB_HOOK_BIN  path to rmb binary (default: local repo bin, built on demand)
 *   RMB_URL       target API (read by rmb hook-submit)
 */

import { existsSync } from "node:fs";
import type { Plugin } from "@opencode-ai/plugin";

const RMB_REPO = "/Users/liguanghui/Virginia/colinleefish/rmb";
const RMB_BIN = `${RMB_REPO}/bin/rmb`;

type SessionMessage = {
	info: { id?: string; role: "user" | "assistant" };
	parts: Array<{ type: string; text?: string }>;
};

// Skip re-uploading the same assistant reply when OpenCode emits duplicate
// idle signals for one settled turn.
const lastUploadedAssistantID = new Map<string, string>();

let resolvedRmbBin: string | undefined;

async function ensureRmbBin(): Promise<string> {
	if (resolvedRmbBin) return resolvedRmbBin;

	const fromEnv = process.env.RMB_HOOK_BIN?.trim();
	if (fromEnv) {
		resolvedRmbBin = fromEnv;
		return fromEnv;
	}

	if (!existsSync(RMB_BIN)) {
		const build = Bun.spawn(["make", "build"], {
			cwd: RMB_REPO,
			stdout: "pipe",
			stderr: "pipe",
		});
		const exitCode = await build.exited;
		if (exitCode !== 0) {
			const errText = await new Response(build.stderr).text();
			throw new Error(`rmb build failed: ${errText.trim() || exitCode}`);
		}
		if (!existsSync(RMB_BIN)) {
			throw new Error(`rmb build finished but ${RMB_BIN} is missing`);
		}
	}

	resolvedRmbBin = RMB_BIN;
	return RMB_BIN;
}

function extractTextParts(parts: SessionMessage["parts"]): string {
	return parts
		.filter((part) => part.type === "text" && part.text?.trim())
		.map((part) => part.text!.trim())
		.join("\n");
}

function findLastTurn(messages: SessionMessage[]): {
	user: string;
	assistant: string;
	assistantMessageID: string;
} {
	let user = "";
	let assistant = "";
	let assistantMessageID = "";
	for (const message of messages) {
		const text = extractTextParts(message.parts);
		if (!text) continue;
		if (message.info.role === "user") user = text;
		if (message.info.role === "assistant") {
			assistant = text;
			assistantMessageID = message.info.id?.trim() ?? "";
		}
	}
	return { user, assistant, assistantMessageID };
}

function sessionIDFromEvent(event: { type: string; properties?: Record<string, unknown> }): string | null {
	const props = event.properties;
	if (!props) return null;
	const id = props.sessionID ?? props.session_id;
	return typeof id === "string" && id.trim() ? id.trim() : null;
}

function isIdleEvent(event: { type: string; properties?: Record<string, unknown> }): boolean {
	// OpenCode emits session.status idle and the deprecated session.idle for the
	// same settle; handling both uploads every turn twice.
	if (event.type !== "session.status") return false;
	const status = event.properties?.status as { type?: string } | undefined;
	return status?.type === "idle";
}

async function fetchSessionMessages(
	client: { session: { messages: (opts: Record<string, unknown>) => Promise<{ data?: SessionMessage[] }> } },
	sessionID: string,
	directory: string,
): Promise<SessionMessage[]> {
	for (const path of [{ sessionID }, { id: sessionID }]) {
		try {
			const response = await client.session.messages({
				path,
				query: { directory },
			});
			if (Array.isArray(response.data)) return response.data;
		} catch {
			// try the other path key shape (SDK v1 vs v2)
		}
	}
	return [];
}

async function submitHook(payload: Record<string, unknown>): Promise<void> {
	const rmbBin = await ensureRmbBin();
	const json = JSON.stringify(payload);
	const proc = Bun.spawn([rmbBin, "hook-submit", "--source=opencode"], {
		stdin: new Blob([json]),
		stdout: "ignore",
		stderr: "pipe",
		env: { ...process.env },
	});
	const exitCode = await proc.exited;
	if (exitCode !== 0) {
		const errText = await new Response(proc.stderr).text();
		throw new Error(errText.trim() || `rmb hook-submit exited ${exitCode}`);
	}
}

const server: Plugin = async ({ client, directory }) => {
	await ensureRmbBin();

	return {
		event: async ({ event }) => {
			if (!isIdleEvent(event)) return;

			const sessionID = sessionIDFromEvent(event);
			if (!sessionID) return;

			try {
				const messages = await fetchSessionMessages(client, sessionID, directory);
				const { user, assistant, assistantMessageID } = findLastTurn(messages);
				if (!assistant) return;

				const dedupeKey = assistantMessageID || `${user}\0${assistant}`;
				if (lastUploadedAssistantID.get(sessionID) === dedupeKey) return;
				lastUploadedAssistantID.set(sessionID, dedupeKey);

				const home = process.env.HOME?.trim();
				const sessionDBPath = home ? `${home}/.local/share/opencode/opencode.db` : undefined;

				await submitHook({
					agent: "opencode",
					session_id: sessionID,
					last_user_message: user,
					last_assistant_message: assistant,
					session_db_path: sessionDBPath,
					cwd: directory,
					hook_event_name: event.type,
				});
			} catch (error) {
				const message = error instanceof Error ? error.message : String(error);
				try {
					await client.app.log({
						service: "rmb-hook",
						level: "error",
						message: "hook-submit failed",
						extra: { sessionID, error: message },
					});
				} catch {
					console.error(`[rmb-hook] upload failed for ${sessionID}: ${message}`);
				}
			}
		},
	};
};

export const RmbHook = server;

export default {
	id: "rmb-hook",
	server,
};
