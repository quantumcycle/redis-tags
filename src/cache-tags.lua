#!lua name=redis_tags

local function inANotInB(a, b)
  local ai = {}
  for i,v in pairs(a) do ai[v]=true end
  for i,v in pairs(b) do 
      if ai[v]~=nil then  
        ai[v] = nil
      end
  end
  return ai
end


local function set(keys, args)
  local hash = keys[1]  -- Get the key name
  local value = table.remove(args,1) -- First arg is the value
  local ttl = tonumber(table.remove(args,1)) -- Second arg is the ttl
  local time = tonumber(redis.call('TIME')[1])  -- Get the current time from the Redis server
  local expected_expiration = time + ttl

  -- Set the value (with optional TTL)
  if ttl == -1 then
    redis.call('SET', hash, value)
    -- no TTL, use a big number for expected expiration (25 years)
    expected_expiration = time + (31536000 * 25)
  else
    redis.call('SET', hash, value, "EX", ttl)
  end

  -- remove this key from old tags
  local old_tags = redis.call('SMEMBERS', hash .. "__rt_tags")
  local tags_to_remove = inANotInB(old_tags, args)
  for k,v in pairs(tags_to_remove) do
    redis.call('ZREM', "__rt_c_tag_" .. k, hash)
    redis.call('SREM', "__rt_tag_" .. k, hash)
  end

  -- save the list of tags
  redis.call('DEL', hash .. "__rt_tags")
  redis.call('SADD', hash .. "__rt_tags", unpack(args))
  if ttl ~= -1 then
    redis.call('EXPIRE', hash .. "__rt_tags", ttl)
  end

  -- Iterate on tags
  for i,tag in pairs(args) do
    -- create a sorted set for tags cleanup
    redis.call('ZADD', "__rt_c_tag_" .. tag, expected_expiration, hash)

    -- create a set for querying
    redis.call('SADD', "__rt_tag_" .. tag, hash)
  end
end

redis.register_function('rt_set', set)

local function del_by_tags(keys, args)
  local tags = {}
  for i,tag in pairs(args) do
    tags[i] = "__rt_tag_" .. tag
  end

  local to_delete = redis.call('SINTER', unpack(tags))

  local deleted = 0
  for i,hash in pairs(to_delete) do
    local keys_deleted = redis.call('DEL', hash)
    redis.call('DEL', hash .. "__rt_tags")
    deleted = deleted + tonumber(keys_deleted)
  end

  return deleted

end

redis.register_function('rt_del_by_tags', del_by_tags)

local function get_keys_by_tags(keys, args)
  local tags = {}
  for i,tag in pairs(args) do
    tags[i] = "__rt_tag_" .. tag
  end

  return redis.call('SINTER', unpack(tags))

end

redis.register_function('rt_get_keys_by_tags', get_keys_by_tags)

local function get_tags(keys, args)
  local pattern = table.remove(args,1)
  
  local tags = {}
  local index = 1
  local cursor_position = 0
  repeat
    local cursor = redis.call('SCAN', cursor_position, 'MATCH', '__rt_tag_' .. pattern, 'COUNT', 1000, 'TYPE', 'set')
    for i,tag in pairs(cursor[2]) do
      -- remmove the '__rt_tag_' prefix
      tags[index] = string.sub(tag, 10)
      index = index + 1
    end
    cursor_position = tonumber(cursor[1])
  until cursor_position == 0

  return tags
end

redis.register_function{
  function_name = 'rt_get_tags',
  callback = get_tags,
  flags = { 'no-writes' }
}

local function cleanup_tag(keys, args)
  local tag = table.remove(args,1)
  local time = tonumber(redis.call('TIME')[1])  -- Get the current time from the Redis server

  --use the sorted sets to get all keys that are supposed to be expired
  local expired_keys = redis.call('ZRANGE', '__rt_c_tag_' .. tag, '-inf', time, 'BYSCORE')
  
  -- check if the key still exist, and if not, remove it from the tag
  for i,hash in pairs(expired_keys) do
    if redis.call('EXISTS', hash) == 0 then
      redis.call('ZREM', '__rt_c_tag_' .. tag, hash)
      redis.call('SREM', '__rt_tag_' .. tag, hash)
    end
  end

  -- if the tag is empty, remove it
  if redis.call('SCARD', '__rt_tag_' .. tag) == 0 then
    redis.call('DEL', '__rt_tag_' .. tag)
    redis.call('DEL', '__rt_c_tag_' .. tag)
  end
end

redis.register_function('rt_cleanup_tag', cleanup_tag)