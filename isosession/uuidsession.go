package isosession

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type UUIDStore struct {
	// Allowed session duration
	// Optional. Default value 24 * time.Hour
	Expiration time.Duration

	// KeyGenerator generates the session key.
	// Optional. Default value utils.UUIDv4
	KeyGenerator func() string

	QueryName string

	// Storage interface to store the session data
	// Optional. Default value memory.New()
	Storage map[string]*UUIDSession
}

type sessIDBody struct {
	ID []string `json:"sessid" form:"sessid"`
}

func (s *UUIDStore) Get(c *fiber.Ctx) (*UUIDSession, error) {
	id := c.Query(s.QueryName)
	if len(id) == 0 {
		var idBody sessIDBody
		if err := c.BodyParser(&idBody); err != nil {
			return nil, err
		}
		if len(idBody.ID) > 0 {
			id = idBody.ID[0]
		}
	}
	var sess *UUIDSession
	var ok bool
	sess, ok = s.Storage[id]
	if ok {
		if sess.fresh {
			sess.fresh = false
		}
		return sess, nil
	}
	sess = &UUIDSession{
		id:    s.KeyGenerator(),
		fresh: true,
		ctx:   c,
		data:  make(map[string]interface{}),
	}
	s.Storage[sess.id] = sess
	return sess, nil
}

type UUIDSession struct {
	id    string
	fresh bool
	ctx   *fiber.Ctx
	data  map[string]interface{}
}

func (s *UUIDSession) ID() string {
	return s.id
}

func (s *UUIDSession) Get(key string) interface{} {
	item, ok := s.data[key]
	if !ok {
		return nil
	}
	return item
}

func (s *UUIDSession) Set(key string, val interface{}) {
	s.data[key] = val
}

func (s *UUIDSession) Delete(key string) {
	delete(s.data, key)
}

func (s *UUIDSession) Destroy() error {
	s.data = map[string]interface{}{}
	return nil
}

func newUUIDSession(config ...UUIDStore) *UUIDStore {
	if len(config) < 1 {
		return &UUIDStore{
			Expiration:   24 * time.Hour,
			KeyGenerator: utils.UUIDv4,
			QueryName:    "sessid",
			Storage:      make(map[string]*UUIDSession),
		}
	}

	cfg := config[0]

	if int(cfg.Expiration.Seconds()) <= 0 {
		cfg.Expiration = 24 * time.Hour
	}
	if cfg.KeyGenerator == nil {
		cfg.KeyGenerator = utils.UUIDv4
	}
	return &cfg
}
