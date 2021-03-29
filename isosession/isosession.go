package isosession

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

var (
	cookieSession = session.New()
	uuidSession   = newUUIDSession()
)

type IsoStore struct {
	cookieStore *session.Store
	uuidStore   *UUIDStore
}

func (s *IsoStore) Get(c *fiber.Ctx) (*session.Session, *UUIDSession, error) {
	uuidSess, err := s.uuidStore.Get(c)
	if err != nil {
		return nil, nil, err
	}
	cookieSess, err := s.cookieStore.Get(c)
	if err != nil {
		return nil, nil, err
	}
	return cookieSess, uuidSess, nil
}

func NewStore() *IsoStore {
	return &IsoStore{
		cookieSession,
		uuidSession,
	}
}
