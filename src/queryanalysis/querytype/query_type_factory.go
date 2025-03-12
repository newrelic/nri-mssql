package querytype

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownQueryType = errors.New("unknown query type")
)

// Factory to create the appropriate QueryType
func CreateQueryType(queryType string) (QueryType, error) {
	switch queryType {
	case "slowQueries":
		return &SlowQueryType{}, nil
	case "waitAnalysis":
		return &WaitQueryType{}, nil
	case "blockingSessions":
		return &BlockingSessionsType{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownQueryType, queryType)
	}
}
