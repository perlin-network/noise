package skademlia

import (
	"encoding/binary"
	"fmt"
	"github.com/perlin-network/noise/payload"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
)

var (
	publicKey1 = []byte("12345678901234567890123456789012")
	publicKey2 = []byte("12345678901234567890123456789011")
	publicKey3 = []byte("12345678901234567890123456789013")
	address    = "localhost:12345"

	id1 = NewID(address, publicKey1)
	id2 = NewID(address, publicKey2)
	id3 = NewID(address, publicKey3)
)

func TestNewID(t *testing.T) {
	t.Parallel()

	hash := blake2b.Sum256(publicKey1)
	assert.EqualValues(t, hash[:], id1.Hash())
	assert.Equal(t, address, id1.address)
}

func TestString(t *testing.T) {
	t.Parallel()

	want := "localhost:12345(3132333435363738)(492c7f5c8f125366)"

	assert.Equal(t, want, id1.String())
}

func TestEquals(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, id1, id2)
	assert.False(t, id1.Equals(id2))
	assert.True(t, id1.Equals(id1))
	assert.False(t, id1.Equals(nil))
}

func noTestXor(t *testing.T) {
	type args struct {
		a []byte
		b []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "id1 xor id2",
			args: args{
				a: id1.PublicID(),
				b: id2.PublicID(),
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name: "id1 xor id3",
			args: args{
				a: id1.Hash(),
				b: id3.Hash(),
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name: "calculated hash",
			args: args{
				a: publicKey1,
				b: publicKey3,
			},
			want: func() []byte {
				publicKey1Hash := blake2b.Sum256(publicKey1)
				publicKey3Hash := blake2b.Sum256(publicKey3)
				result := make([]byte, len(publicKey3Hash))
				for i, b := range publicKey1Hash {
					result[i] = b ^ publicKey3Hash[i]
				}
				return result
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xor(tt.args.a, tt.args.b)
			assert.Equalf(t, got, tt.want, "xor() = %v, want %v", got, tt.want)
		})
	}
}

func TestReadWrite(t *testing.T) {
	t.Parallel()

	testCases := []ID{
		id1,
		id2,
		id3,
	}
	for i, id := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			wrote := id.Write()
			assert.True(t, len(wrote) > len(id.address), "bytes should not be empty")
			placeholder := ID{}
			assert.Falsef(t, id.Equals(placeholder), "Expected not equal %v vs %v", id, placeholder)
			msg, err := placeholder.Read(payload.NewReader(payload.NewWriter(wrote).Bytes()))
			assert.Nil(t, err)
			assert.Truef(t, id.Equals(msg.(ID)), "Expected equal %v vs %v", id, msg)
		})
	}

	// bad
	{
		_, err := id1.Read(payload.NewReader([]byte("bad")))
		assert.NotNil(t, err)
	}
}

func TestPrefixLen(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		publicKeyHash uint32
		expected      int
	}{
		{1, 7},
		{2, 6},
		{4, 5},
		{8, 4},
		{16, 3},
		{32, 2},
		{64, 1},
	}
	for _, tt := range testCases {
		publicKey := make([]byte, 4)
		binary.LittleEndian.PutUint32(publicKey, tt.publicKeyHash)
		assert.Equalf(t, prefixLen(publicKey), tt.expected, "PrefixLen() expected: %d, value: %d", tt.expected, prefixLen(publicKey))
	}
}

func TestPrefixDiff(t *testing.T) {
	t.Parallel()

	a := []byte("aa")
	b := []byte("ab")
	c := []byte("1e")

	key1 := []byte("2b56bb7556eaa58d2253d33b34d7ce869c54bb3c946164f6b73adc378cb9eccab37a3bf66608246c5791ebd19bd25169f6b243a6668c6635b0b4bc43474b6dbd")
	key2 := []byte("2b56as84a56a4e5714b0729019a489521199557143ade85e6e6540d90ac80c6578de0d25fdc274cdff7614dc457333fb7738e29f567e4865f453e2e57c180e67")

	tests := []struct {
		a    []byte
		b    []byte
		n    int
		want int
	}{
		{a, b, 0, 0},
		{a, b, 8, 0},
		{a, b, 9, 0},
		{a, b, 14, 0},
		{a, b, 15, 1},
		{a, b, 16, 2},
		{a, c, 8, 2},
		{a, c, 14, 3},
		{a, c, 16, 3},
		{key1, key2, 192, 52},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			diff := prefixDiff(tt.a, tt.b, tt.n)
			assert.Equal(t, tt.want, diff)
		})
	}
}
