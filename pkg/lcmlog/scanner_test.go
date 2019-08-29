package lcmlog

import (
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
)

func TestScanner_Scan_Testdata(t *testing.T) {
	f, err := os.Open("testdata/lcmlog.00")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()
	sc := NewScanner(f)
	var i int32
	for sc.Scan() {
		require.Equal(t, []byte("test"), sc.Message().Channel)
		require.Equal(t, uint64(i), sc.Message().EventNumber)
		ts := &timestamp.Timestamp{}
		require.NoError(t, proto.Unmarshal(sc.Message().Data, ts))
		require.Equal(t, &timestamp.Timestamp{Nanos: i}, ts)
		i++
	}
}
