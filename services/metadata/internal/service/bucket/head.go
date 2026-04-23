package bucket

import "context"

func (s *bucketService) HeadBucket(ctx context.Context, name string) (bool, string, string, error) {
	return s.repo.Head(ctx, name)
}
