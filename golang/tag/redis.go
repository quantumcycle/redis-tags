package tag

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func SetWithTags(ctx context.Context, c *redis.Client, key string, value interface{}, ttlSeconds int, tags ...string) error {
	params := make([]interface{}, 0, 2+len(tags))
	params = append(params, value)
	params = append(params, ttlSeconds)
	for _, tag := range tags {
		params = append(params, tag)
	}
	err := c.FCall(ctx, "rt_set", []string{key}, params...).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}

func DelByTags(ctx context.Context, c *redis.Client, tags ...string) (int, error) {
	params := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		params = append(params, tag)
	}
	cmd := c.FCall(ctx, "rt_del_by_tags", nil, params...)
	if err := cmd.Err(); err != nil && err != redis.Nil {
		return 0, err
	}
	return int(cmd.Val().(int64)), nil
}

func GetKeysByTags(ctx context.Context, c *redis.Client, tags ...string) ([]string, error) {
	params := make([]interface{}, 0, len(tags))
	for _, tag := range tags {
		params = append(params, tag)
	}
	cmd := c.FCall(ctx, "rt_get_keys_by_tags", nil, params...)
	if err := cmd.Err(); err != nil && err != redis.Nil {
		return nil, err
	}
	keys, err := cmd.StringSlice()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	return keys, nil
}

func GetTags(ctx context.Context, c *redis.Client, pattern string) ([]string, error) {
	params := make([]interface{}, 0, 1)
	params = append(params, pattern)
	cmd := c.FCall(ctx, "rt_get_tags", nil, params...)
	res, err := cmd.Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	tags := make([]string, 0, len(res.([]interface{})))
	for _, tag := range res.([]interface{}) {
		tags = append(tags, tag.(string))
	}
	return tags, nil
}

func CleanupTag(ctx context.Context, c *redis.Client, tag string) error {
	err := c.FCall(ctx, "rt_cleanup_tag", nil, tag).Err()
	if err != nil && err != redis.Nil {
		return err
	}
	return nil
}
