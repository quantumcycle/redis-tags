package test

import (
	"context"
	"fmt"
	"redis-tags-tests/tag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/redis/go-redis/v9"
)

func TestDeletesKeysMatchingTags(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
	})

	rdb.FlushAll(ctx)

	err := tag.SetWithTags(ctx, rdb, "key-1-123", "value", 60, "tag:1", "tag:2", "tag:3")
	if err != nil {
		t.Error(err)
	}

	err = tag.SetWithTags(ctx, rdb, "key-2-23", "value", 60, "tag:2", "tag:3")
	if err != nil {
		t.Error(err)
	}

	err = tag.SetWithTags(ctx, rdb, "key-3-3", "value", 60, "tag:3")
	if err != nil {
		t.Error(err)
	}

	err = tag.SetWithTags(ctx, rdb, "key-4-3", "value", 60, "tag:3")
	if err != nil {
		t.Error(err)
	}

	//Delete all with 3 tags, should only delete key-1-123
	deleted, err := tag.DelByTags(ctx, rdb, "tag:1", "tag:2", "tag:3")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 1, deleted)
	keys := rdb.Keys(ctx, "key-1-123")
	assert.Equal(t, 0, len(keys.Val()))

	//Delete all a single tag, should key-2-23, key-3-3, key-4-3
	deleted, err = tag.DelByTags(ctx, rdb, "tag:3")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 3, deleted)
	keys = rdb.Keys(ctx, "key-2-23")
	assert.Equal(t, 0, len(keys.Val()))
	keys = rdb.Keys(ctx, "key-3-3")
	assert.Equal(t, 0, len(keys.Val()))
	keys = rdb.Keys(ctx, "key-4-3")
	assert.Equal(t, 0, len(keys.Val()))
}

func TestDeletesEverythingWhenExpired(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
	})

	rdb.FlushAll(ctx)

	//Create 100 keys and assign them to 7 possible tags
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		var tags []string
		tagVal := i % 5
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+1))
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+2))
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+3))
		err := tag.SetWithTags(ctx, rdb, key, "value", 1, tags...)
		if err != nil {
			t.Error(err)
		}
	}

	//Wait for the last key to expire
	time.Sleep(1 * time.Second)

	//Cleanup all keys
	tags, err := tag.GetTags(ctx, rdb, "tag:*")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 7, len(tags))

	for _, myTag := range tags {
		err := tag.CleanupTag(ctx, rdb, myTag)
		if err != nil {
			t.Error(err)
		}
	}

	//Wait for the last set to be deleted
	time.Sleep(1 * time.Second)

	keys := rdb.Keys(ctx, "*")
	assert.Equal(t, 0, len(keys.Val()))
}

func TestGetTagsShouldReturnAllTags(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
	})

	rdb.FlushAll(ctx)

	var tags []string
	for i := 0; i < 1005; i++ {
		tags = append(tags, fmt.Sprintf("tag:%d", i))
	}

	err := tag.SetWithTags(ctx, rdb, "key-1", "value", 1, tags...)
	if err != nil {
		t.Error(err)
	}

	tags, err = tag.GetTags(ctx, rdb, "tag:*")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 1005, len(tags))
}

func TestBenchmark(t *testing.T) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
	})

	rdb.FlushAll(ctx)

	start := time.Now()

	//Add a 50,000 thousand keys with 3 tags each
	//1002 tags total
	for i := 0; i < 50000; i++ {
		key := fmt.Sprintf("key-%d", i)
		var tags []string
		tagVal := i % 1000
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+1))
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+2))
		tags = append(tags, fmt.Sprintf("tag:%d", tagVal+3))
		err := tag.SetWithTags(ctx, rdb, key, "value", 1, tags...)
		if err != nil {
			t.Error(err)
		}
	}

	//Flush all the tags one by one
	for i := 1; i <= 1002; i++ {
		_, err := tag.DelByTags(ctx, rdb, fmt.Sprintf("tag:%d", i))
		if err != nil {
			t.Error(err)
		}
	}

	elapsed := time.Since(start)

	//Fail test if it takes more than 30 seconds
	assert.Less(t, elapsed.Seconds(), 30.0)

	//Wait for the last set to be deleted
	time.Sleep(1 * time.Second)

	keys := rdb.Keys(ctx, "key-*")
	assert.Equal(t, 0, len(keys.Val()))
}
