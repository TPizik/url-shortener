package services

type IStorage interface {
	Get(key string) (string, error)
	Add(url string) string
}

type Service struct {
	storage IStorage
}

func NewService(storage IStorage) Service {
	return Service{
		storage: storage,
	}
}

func (s *Service) CreateRedirect(key string) string {
	return s.storage.Add(key)
}

func (s *Service) GetURLByKey(key string) (string, error) {
	return s.storage.Get(key)
}
