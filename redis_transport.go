package mercure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	redis "github.com/go-redis/redis/v8"
)

func init() { //nolint:gochecknoinits
	RegisterTransportFactory("redis", NewRedisTransport)
}

// RedisTransport implements the TransportInterface using the Redis database.
type RedisTransport struct {
	sync.RWMutex
	closed           chan struct{}
	closedOnce       sync.Once
	client           *redis.Client
	lastEventID      string
	logger           Logger
	subscribers      map[*Subscriber]struct{}
}

// NewRedisTransport create a new redisTransport.
func NewRedisTransport(u *url.URL, l Logger, tss *TopicSelectorStore) (Transport, error) {
	var err error

	l.Debug("new redis transport")

	path := u.Path // absolute path (redis:///path.db)
	if path == "" {
		path = u.Host // relative path (redis://path.db)
	}
	if path == "" {
		return nil, &ErrTransport{u.Redacted(), "missing path", err}
	}

	client := redis.NewClient(&redis.Options{
        Addr:     path,
        Password: "", // no password set
        DB:       0,  // use default DB
    })

	return &RedisTransport{
		closed:           make(chan struct{}),
		client:           client,
		logger:           l,
		subscribers:      make(map[*Subscriber]struct{}),
		lastEventID:      "42",
	}, nil
}

// Dispatch dispatches an update to all subscribers.
func (t *RedisTransport) Dispatch(update *Update) error {

	t.logger.Debug("dispatch")

	select {
	case <-t.closed:
		return ErrClosedTransport
	default:
	}

	AssignUUID(update)

	t.Lock()
	defer t.Unlock()

	for subscriber := range t.subscribers {
		if !subscriber.Dispatch(update, false) {
			delete(t.subscribers, subscriber)
		}
	}

	return nil
}

func (t *RedisTransport) persist(s *Subscriber) error {

	t.logger.Debug("Persist")

	subJson, erro := json.Marshal(s)

	if erro != nil {
		t.logger.Error("Error while marshalling a subscriber")
        return erro
    }

	var ctx = context.Background()
	err := t.client.LPush(ctx, "subscribers", subJson).Err()

    if err != nil {
		t.logger.Error("Error while setting a subscriber")
        return err
    }

	return nil
}

// AddSubscriber adds a new subscriber to the transport.
func (t *RedisTransport) AddSubscriber(s *Subscriber) error {

	t.logger.Debug("Add a subscriber")

	t.Lock()
	defer t.Unlock()

	select {
	case <-t.closed:
		return ErrClosedTransport
	default:
	}

	t.subscribers[s] = struct{}{}
	t.persist(s)

	return nil
}

// GetSubscribers get the list of active subscribers.
func (t *RedisTransport) GetSubscribers() (string, []*Subscriber, error) {

	t.logger.Debug("Get subscribers")

	t.RLock()
	defer t.RUnlock()

	t.logger.Info("after for")

	var ctx = context.Background()
	val, err := t.client.LRange(ctx, "subscribers", 0, 100).Result()
    if err != nil {
		t.logger.Error("error while getting subscribers")
    }

	subscribers := make([]*Subscriber, len(val))

	for i,subscriber := range val {
		fmt.Println(i)
		fmt.Println(subscriber)

		var subUnmarshalled *Subscriber
		if err := json.Unmarshal([]byte(subscriber), &subUnmarshalled); err != nil {
			fmt.Errorf("unable to unmarshal subscriber: %w", err)
		}

		subscribers[i] = subUnmarshalled
	}

	return t.lastEventID, subscribers, nil
}

// Close closes the Transport.
func (t *RedisTransport) Close() (err error) {

	t.logger.Info("Close")

	t.closedOnce.Do(func() {
		close(t.closed)

		t.Lock()
		for subscriber := range t.subscribers {
			subscriber.Disconnect()
			delete(t.subscribers, subscriber)
		}
		t.Unlock()

		err = t.client.Close()
	})

	if err == nil {
		return nil
	}

	return fmt.Errorf("unable to close Redis DB: %w", err)
}

// Interface guards.
var (
	_ Transport            = (*RedisTransport)(nil)
	_ TransportSubscribers = (*RedisTransport)(nil)
)
