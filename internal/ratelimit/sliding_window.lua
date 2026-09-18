-- Sliding-window-log rate limiter.
--
-- KEYS[1] = the rate limit key (e.g. "ratelimit:1.2.3.4")
-- ARGV[1] = now, in nanoseconds
-- ARGV[2] = window size, in nanoseconds
-- ARGV[3] = max requests allowed per window
-- ARGV[4] = unique member id for this request (avoids collisions when
--           two requests land in the same nanosecond)
--
-- Runs as a single Lua script so the "count, then maybe add" sequence
-- is atomic — without this, two concurrent requests could both read a
-- count of (limit - 1) and both be admitted, letting a client burst
-- past the limit.
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

redis.call('ZREMRANGEBYSCORE', key, '-inf', now - window)

local count = redis.call('ZCARD', key)
if count >= limit then
    return 0
end

redis.call('ZADD', key, now, member)
redis.call('PEXPIRE', key, math.ceil(window / 1e6) + 1000)
return 1
