import fs from "fs/promises";
import NodeCache from "node-cache";
import path from "path";
import pino from "pino";
import { z } from "zod";

// ---------------------- Logger ----------------------
const logger = pino({
  level: process.env.LOG_LEVEL || "info",
  base: { service: "sprint-update-state" },
});

// ---------------------- Cache ----------------------
const cacheTTL = parseInt(process.env.UPDATE_CACHE_TTL || "300", 10);
const cache = new NodeCache({ stdTTL: cacheTTL, checkperiod: 60 });

// ---------------------- Schema ----------------------
export const UpdateStateSchema = z.object({
  version: z.string().min(1, "Version must be a non-empty string"),
  last_updated: z.string().refine(val => !isNaN(Date.parse(val)), {
    message: "Must be a valid ISO 8601 datetime",
  }),
  rollback: z.boolean(),
});

export type UpdateState = z.infer<typeof UpdateStateSchema>;

// ---------------------- Helpers ----------------------
function resolveStateFilePath(): string {
  const statePath = process.env.SPRINT_STATE_FILE || path.join(process.cwd(), "data", "update_state.json");
  const resolved = path.normalize(path.resolve(statePath));

  if (!resolved.startsWith(process.cwd()) && !resolved.startsWith("/var/lib/sprint")) {
    throw new Error(`Invalid state file path: ${resolved}`);
  }
  return resolved;
}

// ---------------------- Main API ----------------------
export async function getUpdateState(): Promise<UpdateState> {
  const cacheKey = "update_state";
  const cached = cache.get<UpdateState>(cacheKey);
  if (cached) {
    logger.debug("Returning cached update state");
    return cached;
  }

  const resolvedPath = resolveStateFilePath();
  logger.debug({ path: resolvedPath }, "Reading update state file");

  let data: string;
  try {
    data = await fs.readFile(resolvedPath, "utf-8");
  } catch (e: any) {
    if (e.code === "ENOENT") {
      logger.error({ path: resolvedPath }, "State file not found");
      throw new Error("State file not found");
    }
    throw e;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(data);
  } catch (e: any) {
    logger.error({ error: e.message }, "Failed to parse state file JSON");
    throw new Error("Invalid state file format");
  }

  const state = UpdateStateSchema.safeParse(parsed);
  if (!state.success) {
    logger.error({ errors: state.error.errors }, "State validation failed");
    throw new Error("Invalid state data");
  }

  cache.set(cacheKey, state.data);
  logger.info({ version: state.data.version }, "Successfully retrieved update state");
  return state.data;
}
