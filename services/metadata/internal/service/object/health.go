package object

import "context"

func (s *objectService) HealthCheck(ctx context.Context) (bool, bool) {
	pgOK := s.pgClient.DB().Ping(ctx) == nil
	kafkaOK := s.saramaClient.RefreshMetadata() == nil
	return pgOK, kafkaOK
}
