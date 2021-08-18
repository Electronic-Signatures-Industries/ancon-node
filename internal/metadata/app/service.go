package app

type MetadataRepository interface {
	Add(Metadata, ...interface{}) error
}

type MetadataService struct {
	repository MetadataRepository
}

func (s *MetadataService) Add(data Metadata, options ...interface{}) error {
	return s.repository.Add(data, options...)
}
