// Lightweight Redis-backed per-minute limiter with graceful fallback to allow-all.
// Uses dynamic require so ioredis is optional in development.
/* eslint-disable @typescript-eslint/no-explicit-any */
declare const require: any;

let redisClient: any = (global as any).__redisClient;
async function getRedis() {
  if (redisClient) return redisClient;
  try {
    const IORedis = require('ioredis');
    redisClient = new IORedis(process.env.REDIS_URL);
    (global as any).__redisClient = redisClient;
    return redisClient;
  } catch (err) {
    // ioredis not installed or REDIS_URL not set
    return null;
  }
}

/**
 * Simple per-minute counter using Redis INCR + EXPIRE. If Redis isn't available,
 * this function returns allowed: true so the system continues to work.
 */
export async function isRequestAllowed(token: string, requestsPerMinute: number): Promise<{ allowed: boolean; reason?: string }> {
  if (!process.env.REDIS_URL) {
    return { allowed: true, reason: 'No REDIS_URL' };
  }

  const redis = await getRedis();
  if (!redis) return { allowed: true, reason: 'Redis client not available' };

  const windowSeconds = 60;
  const bucket = Math.floor(Date.now() / 1000 / windowSeconds);
  const key = `rl:${token}:${bucket}`;

  try {
    const cnt = await redis.incr(key);
    if (cnt === 1) {
      await redis.expire(key, windowSeconds + 1);
    }
    if (cnt > requestsPerMinute) {
      return { allowed: false, reason: 'Rate limit exceeded' };
    }
    return { allowed: true };
  } catch (err) {
    console.error('[rateLimiter] Redis error:', err);
    return { allowed: true, reason: 'Redis error' };
  }
}

export default { isRequestAllowed };
