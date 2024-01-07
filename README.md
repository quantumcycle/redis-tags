# redis-tags

A collection of redis functions to tags entries and remove them based on their tags

## How to install this

Since Redis 7.x, you can have the equivalent of a database stored procedure in Redis. You can take a lua script, and "install" it in your redis instance so you can later call it. This is the preferred method of installing this project in your redis instance.

To install this library in your redis instance, clone this repo, `cd` into it, and use the following command
```
cat ./src/cache-tags.lua | redis-cli -x FUNCTION LOAD REPLACE
```
If you need to update to the latest version, just `git pull` the repo, and run the command again.

From this point you will be able to call the different functions using the `FCALL` redis operation. For the full list of commands, see below.

### Support for Redis 6 and lower

If you're using Redis 6 or lower, you can still use this project but it's gonna require more work. You're going to have to extract the Lua scripts in the `src` folder, and integrate them in your codebase as helper functions that will call Redis `EVAL` function.

# Functions

All functions are prefixed by `rt_` for Redis Tags. There is no namespacing of functions in Redis, so this prefix serves to avoid collision with other potential user defined functions.

## Create/Update a key with tags `rt_set`

This will create or update a key with a value (equivalent to `SET k v`), and attach that key to 1-N tags.

```
FCALL rt_set 1 <key> <value> <ttl> <tag1> <tag2> <tag3>...
```
Params:
* key: name of the key
* value: value of the key
* ttl: time to live in seconds. For keys without TTL, use `-1`
* tags: a list of tags for the key

## Delete keys by tags `rt_del_by_tags`

This will delete all the keys matching all the tags. This will delete the keys that have ALL the tags passed as argument.

```
FCALL rt_del_by_tags 0 <tag1> <tag2> <tag3>...
```
Params:
* tags: a list of tags. the deleted keys must have all the specified tags.

## Get keys by tags `rt_del_by_tags`

This will return all the keys matching all the tags. This is an intersection of all the tags, not a union.

```
FCALL rt_get_keys_by_tags 0 <tag1> <tag2> <tag3>...
```
Params:
* tags: a list of tags. the returned keys must have ALL the specified tags.

# Cleanup job

Since we're creating keys to track the tags, and since keys can expire naturally, we need a way to cleanup the tags sets, otherwise they would just grow forever. To achieve this, you will need to have some kind of cron job that will periodically cleanup the tags.

There is 2 functions that are required for cleanup.

## Getting all the tags

The first function is `rt_get_tags`. This function returns all the tags we're tracking in redis.
```
FCALL rt_get_tags 0 <pattern>
```
Params:
* pattern: prefix pattern for the tags to recover. To get all the tags, use `*`
* ttl: time to live in seconds. For keys without TTL, use `-1`

## Doing the cleanup for a specific tag

The second function is `rt_cleanup_tag`. This function receives a tag and does the cleanup.

```
FCALL cleanup_tag 0 <tag>
```
Params:
* tag: the value of the tag to cleanup

## Cron job

You need to combine the 2 functions and call `rt_cleanup_tag` for each tag returned by `rt_get_all_tags`. There is multiple ways to achieve this depending on your programming language. The volume of tags is gonna impact how you do it.