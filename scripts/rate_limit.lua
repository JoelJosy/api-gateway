-- Fetch current token count and last refill timestamp from the client's Redis hash
local data = redis.call('HMGET', KEYS[1], 'tokens', 'last_refill')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local current_time = tonumber(ARGV[3])

-- If the key doesn't exist yet, initialize it to full capacity
if tokens == nil then
    tokens = max_tokens
    last_refill = current_time
else
    -- Calculate how many tokens have accumulated since the last request
    local elapsed = current_time - last_refill
    if elapsed > 0 then
        tokens = math.min(max_tokens, tokens + (elapsed * refill_rate))
        last_refill = current_time
    end
end

-- Check if the user has enough tokens to pass through
if tokens < 1 then
    -- Return 0 => Reject with 429
    return {0, math.floor(tokens)}
else
    -- Deduct one token, save changes back to the hash, and set a safety expiry 
    tokens = tokens - 1
    redis.call('HMSET', KEYS[1], 'tokens', tokens, 'last_refill', last_refill)
    redis.call('EXPIRE', KEYS[1], 3600) -- Clean up memory if inactive for an hour
    
    -- Return 1 => Allow request
    return {1, math.floor(tokens)}
end