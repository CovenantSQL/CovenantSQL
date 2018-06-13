package merkle

import (
	"testing"

	"bytes"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/sqlchain"
)

func TestMergeTwoHash(t *testing.T) {
	Convey("Concatenate of two hash should be equal to the result", t, func() {
		h0, _ := hash.NewHash([]byte{
			62, 111, 80, 14, 71, 212, 231, 131,
			184, 81, 1, 100, 137, 185, 147, 131,
			144, 215, 166, 168, 245, 177, 176, 12,
			149, 171, 64, 220, 24, 240, 0, 185,
		})
		h1, _ := hash.NewHash([]byte{
			215, 69, 48, 107, 167, 190, 161, 249,
			16, 185, 81, 90, 27, 205, 142, 66,
			242, 201, 251, 218, 129, 201, 116, 71,
			55, 247, 82, 142, 190, 50, 97, 17,
		})
		wanted := []byte{
			5, 139, 186, 34, 229, 19, 224, 226,
			79, 42, 32, 250, 185, 95, 234, 201,
			62, 36, 31, 54, 209, 2, 17, 98,
			222, 231, 79, 93, 101, 24, 169, 178,
		}

		So(bytes.Compare(MergeTwoHash(h0, h1).CloneBytes(), wanted), ShouldEqual, 0)
	})
}

type tx struct {
	v []byte
}

func (t tx) Hash() *hash.Hash {
	h := hash.THashH(t.v)
	return &h
}

func TestNewMerkle(t *testing.T) {
	tests := []struct {
		t      []tx
		wanted []byte
	}{
		{
			[]tx{
				tx{[]byte{1, 2, 3, 4, 5, 6}},
				tx{[]byte{9, 8, 6, 5, 3}},
				tx{[]byte{91, 12, 10, 92}},
			},
			[]byte{
				202, 225, 160, 198, 238, 212, 230, 182, 114, 26, 93, 135, 70, 173, 65, 166,
				111, 203, 170, 132, 135, 252, 178, 113, 211, 60, 190, 144, 6, 243, 134, 7,
			},
		},
		{
			[]tx{
				tx{[]byte{1, 2, 3, 4, 5, 6}},
				tx{[]byte{9, 8, 6, 5, 3}},
				tx{[]byte{91, 12, 10, 92}},
				tx{[]byte{0, 0, 0, 0, 0, 0, 0, 0}},
			},
			[]byte{
				71, 14, 13, 160, 40, 17, 240, 13, 252, 166, 143, 68, 177, 180, 15, 193, 177,
				178, 249, 127, 23, 56, 221, 241, 27, 254, 236, 140, 68, 19, 122, 130,
			},
		},
		{
			[]tx{
				tx{[]byte{1, 2, 3, 4, 5, 6}},
				tx{[]byte{9, 8, 6, 5, 3}},
				tx{[]byte{91, 12, 10, 92}},
				tx{[]byte{0, 0, 0, 0, 0, 0, 0, 0}},
				tx{[]byte{2, 0, 23, 120, 3}},
			},
			[]byte{
				183, 1, 159, 134, 148, 115, 20, 37, 124, 7, 96, 98, 172, 8, 178, 234, 33, 184,
				65, 26, 101, 72, 119, 57, 145, 39, 26, 242, 3, 201, 191, 68,
			},
		},
	}
	Convey("Two root hashes should be the same", t, func() {
		for _, t := range tests {
			hashableTx := make([]sqlchain.Hashable, len(t.t))
			for i, t := range t.t {
				hashableTx[i] = t
			}
			merkle := NewMerkle(hashableTx)
			root := merkle.GetRoot()
			So(bytes.Compare(t.wanted, root[:]), ShouldEqual, 0)
		}

	})

}
