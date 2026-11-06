/*
 * Copyright 2026 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types_test

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

const goldenFile = "testdata/marshalhash.golden"

var updateGolden = flag.Bool("update", false, "rewrite "+goldenFile+" with the current MarshalHash output")

// hashedTypes has a value of every type with a generated MarshalHash.
var hashedTypes = []verifier.MarshalHasher{
	// blockproducer/interfaces
	new(pi.AccountNonce), new(pi.TransactionState), new(pi.TransactionType),
	new(pi.TransactionTypeMixin),
	// crypto/verifier
	new(verifier.DefaultHashSignVerifierImpl),
	// proto
	new(proto.AddrAndGas), new(proto.DatabaseID), new(proto.Envelope), new(proto.FindNeighborReq),
	new(proto.FindNeighborResp), new(proto.FindNodeReq), new(proto.FindNodeResp), new(proto.Node),
	new(proto.NodeID), new(proto.NodeKey), new(proto.Peers), new(proto.PeersHeader),
	new(proto.PingReq), new(proto.PingResp), new(proto.RawNodeID), new(proto.ServerRole),
	new(proto.ServerRoles), new(proto.UploadMetricsReq), new(proto.UploadMetricsResp),
	// types
	new(types.Account), new(types.Ack), new(types.AckHeader), new(types.AckResponse),
	new(types.BaseAccount), new(types.Block), new(types.Blocks), new(types.BPBlock),
	new(types.BPHeader), new(types.BPSignedHeader), new(types.CreateDatabase),
	new(types.CreateDatabaseHeader), new(types.CreateDatabaseRequest),
	new(types.CreateDatabaseRequestHeader), new(types.CreateDatabaseResponse),
	new(types.CreateDatabaseResponseHeader), new(types.DropDatabaseRequest),
	new(types.DropDatabaseRequestHeader), new(types.DropDatabaseResponse),
	new(types.GetDatabaseRequest), new(types.GetDatabaseRequestHeader),
	new(types.GetDatabaseResponse), new(types.GetDatabaseResponseHeader), new(types.Header),
	new(types.InitService), new(types.InitServiceResponse), new(types.InitServiceResponseHeader),
	new(types.IssueKeys), new(types.IssueKeysHeader), new(types.MinerIncome), new(types.MinerInfo),
	new(types.MinerKey), new(types.NamedArg), new(types.ProviderProfile), new(types.ProvideService),
	new(types.ProvideServiceHeader), new(types.Query), new(types.QueryAsTx), new(types.QueryKey),
	new(types.QueryType), new(types.Range), new(types.Request), new(types.RequestHeader),
	new(types.RequestPayload), new(types.ResourceMeta), new(types.Response),
	new(types.ResponseHeader), new(types.ResponsePayload), new(types.ResponseRow),
	new(types.ServiceInstance), new(types.SignedAckHeader),
	new(types.SignedCreateDatabaseRequestHeader), new(types.SignedCreateDatabaseResponseHeader),
	new(types.SignedDropDatabaseRequestHeader), new(types.SignedGetDatabaseRequestHeader),
	new(types.SignedGetDatabaseResponseHeader), new(types.SignedHeader),
	new(types.SignedInitServiceResponseHeader), new(types.SignedRequestHeader),
	new(types.SignedResponseHeader), new(types.SignedUpdateServiceHeader),
	new(types.SQLChainProfile), new(types.SQLChainRole), new(types.SQLChainUser), new(types.Status),
	new(types.TokenType), new(types.Transfer), new(types.TransferHeader), new(types.UpdateBilling),
	new(types.UpdateBillingHeader), new(types.UpdatePermission), new(types.UpdatePermissionHeader),
	new(types.UpdateService), new(types.UpdateServiceHeader), new(types.UpdateServiceResponse),
	new(types.UpdateType), new(types.UserArrears), new(types.UserCost), new(types.UserPermission),
	new(types.UserPermissionRole),
}

// filler sets every exported field, element and pointer to a value derived from a counter, so
// that a moved field or a changed encoding changes the bytes. Integers vary their sign and size,
// and strings and byte slices their length, to use the different encodings of each.
type filler struct {
	n   int
	pub *asymmetric.PublicKey
	sig *asymmetric.Signature
}

func newFiller() *filler {
	priv, pub := asymmetric.PrivKeyFromBytes(bytes.Repeat([]byte{1}, 32))
	sig, err := priv.Sign(bytes.Repeat([]byte{2}, 32))
	if err != nil {
		panic(err)
	}
	return &filler{pub: pub, sig: sig}
}

func (f *filler) fill(v reflect.Value) {
	f.n++
	n := int64(f.n)
	// 2 to 304 bytes, across the 5-, 8- and 16-bit length encodings.
	text := fmt.Sprint("s", n) + strings.Repeat("x", f.n%3*150)
	switch v.Type() {
	case reflect.TypeOf(f.pub):
		v.Set(reflect.ValueOf(f.pub))
		return
	case reflect.TypeOf(f.sig):
		v.Set(reflect.ValueOf(f.sig))
		return
	case reflect.TypeOf(time.Time{}):
		v.Set(reflect.ValueOf(time.Unix(1500000000+n, 1000*n).UTC()))
		return
	case reflect.TypeOf((*pi.Transaction)(nil)).Elem():
		tx := new(types.Transfer)
		f.fill(reflect.ValueOf(tx).Elem())
		v.Set(reflect.ValueOf(tx))
		return
	}
	switch v.Kind() {
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
		f.fill(v.Elem())
	case reflect.Struct:
		// Fill the fields in name order: hsp sorts them, so the declaration order is not encoded.
		fields := make([]reflect.StructField, v.NumField())
		for i := range fields {
			fields[i] = v.Type().Field(i)
		}
		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
		for _, field := range fields {
			switch {
			case field.PkgPath != "": // unexported, not encoded
			case field.Tag.Get("hsp") == "v,version":
				// MarshalHash picks the encoding by this field; use the one new values get.
				def := v.Addr().Interface().(interface{ HSPDefaultVersion() int }).HSPDefaultVersion()
				v.FieldByIndex(field.Index).SetInt(int64(def))
			default:
				f.fill(v.FieldByIndex(field.Index))
			}
		}
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes([]byte(text))
			break
		}
		// Slices of pointers get a third element, left nil, which encodes as nil.
		size := 2
		if v.Type().Elem().Kind() == reflect.Ptr {
			size = 3
		}
		v.Set(reflect.MakeSlice(v.Type(), size, size))
		f.fill(v.Index(0))
		f.fill(v.Index(1))
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			f.fill(v.Index(i))
		}
	case reflect.Interface:
		// One value of each type that hsp.AppendIntf encodes on its own, and a string over 64 KiB.
		v.Set(reflect.ValueOf([]interface{}{nil, true, false,
			-n, -n << 40, int(n) << 36, int8(-n), int16(n << 4), int32(-n << 12),
			uint(n), uint8(n), uint16(n << 4), uint32(n << 12), uint64(n) << 40,
			float32(n) + 0.5, float64(n) + 0.5, text, strings.Repeat("y", 1<<16+f.n), []byte(text),
			time.Unix(n, 1000*n).UTC()}))
	case reflect.String:
		v.SetString(text)
	case reflect.Bool:
		v.SetBool(f.n%2 == 0)
		return
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// The sign changes every fourth value, so that each size is used with both signs.
		v.SetInt(f.integer(v.Type().Bits()) * (1 - n/4%2*2))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(f.integer(v.Type().Bits())))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(n) + 0.5)
	default:
		panic("cannot fill " + v.Type().String())
	}
	// A struct is zero only when it has no exported fields, like types.AckResponse.
	if v.Kind() != reflect.Struct && v.IsZero() {
		panic("filled " + v.Type().String() + " with its zero value")
	}
}

// integer returns a non-zero value for an integer of the given width: bytes cycle through 1 to
// 255, wider integers are the counter shifted by 0, 1/4, 1/2 or 3/4 of their width.
func (f *filler) integer(bits int) int64 {
	if bits == 8 {
		return int64(f.n%255 + 1)
	}
	return int64(f.n) << (f.n % 4 * bits / 4)
}

// generatedMethods returns "package.Type.Method" for every MarshalHash method in the
// hsp-generated files of the repository.
func generatedMethods() (names []string, err error) {
	err = filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && d.Name() != ".." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata"):
			return filepath.SkipDir
		case d.IsDir() || !strings.HasSuffix(path, "_gen.go"):
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil || !bytes.Contains(src, []byte("Code generated by github.com/CovenantSQL/HashStablePack")) {
			return err
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, src, 0)
		if err != nil {
			return err
		}
		pkg := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(path, ".."+string(filepath.Separator))))
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !strings.HasPrefix(fn.Name.Name, "MarshalHash") {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			names = append(names, pkg+"."+recv.(*ast.Ident).Name+"."+fn.Name.Name)
		}
		return nil
	})
	return
}

func TestMarshalHashGolden(t *testing.T) {
	Convey("MarshalHash of every generated type matches "+goldenFile, t, func() {
		got := map[string]string{}
		var failed []string
		record := func(name string, marshal func() ([]byte, error)) {
			b, err := marshal()
			if err != nil {
				failed = append(failed, name+": "+err.Error())
			}
			got[name] = fmt.Sprintf("%x", sha256.Sum256(b))
		}
		for _, v := range hashedTypes {
			typ := reflect.TypeOf(v).Elem()
			name := strings.TrimPrefix(typ.PkgPath(), "github.com/CovenantSQL/CovenantSQL/") + "." + typ.Name()
			// The zero value takes the nil, empty and default-version branches.
			record(name+".MarshalHash/zero", reflect.New(typ).Interface().(verifier.MarshalHasher).MarshalHash)
			newFiller().fill(reflect.ValueOf(v).Elem())
			record(name+".MarshalHash", v.MarshalHash)
		}
		// MarshalHash of UpdateBillingHeader runs one of these, as Version says.
		h := new(types.UpdateBillingHeader)
		newFiller().fill(reflect.ValueOf(h).Elem())
		record("types.UpdateBillingHeader.MarshalHasholdver", h.MarshalHasholdver)
		record("types.UpdateBillingHeader.MarshalHash9ef447", h.MarshalHash9ef447)
		So(failed, ShouldBeEmpty)

		generated, err := generatedMethods()
		So(err, ShouldBeNil)
		var uncovered []string
		for _, name := range generated {
			if _, ok := got[name]; !ok {
				uncovered = append(uncovered, name)
			}
		}
		So(uncovered, ShouldBeEmpty)

		var lines []string
		for name, sum := range got {
			lines = append(lines, name+" "+sum)
		}
		sort.Strings(lines)
		if *updateGolden {
			So(os.WriteFile(goldenFile, []byte(strings.Join(lines, "\n")+"\n"), 0644), ShouldBeNil)
		}
		data, err := os.ReadFile(goldenFile)
		So(err, ShouldBeNil)
		want := map[string]string{}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			name, sum, _ := strings.Cut(line, " ")
			want[name] = sum
		}
		// A name here means its encoding changed. If that is intended, rerun with -update.
		var changed []string
		for name, sum := range got {
			if want[name] != sum {
				changed = append(changed, name)
			}
			delete(want, name)
		}
		for name := range want {
			changed = append(changed, name)
		}
		sort.Strings(changed)
		So(changed, ShouldBeEmpty)
	})
}
