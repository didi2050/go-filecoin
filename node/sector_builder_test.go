package node

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	dag "gx/ipfs/QmNUCLv5fmUBuAcwbkt58NQvMcJgd5FPCYV2yNCXq4Wnd6/go-ipfs/merkledag"

	"github.com/filecoin-project/go-filecoin/repo"
	"github.com/filecoin-project/go-filecoin/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sectorDirsForTest = &repo.MemRepo{}

func TestSimple(t *testing.T) {
	require := require.New(t)
	nd := MakeOfflineNode(t)
	sb := requireSectorBuilder(require, nd, 50)
	sector, err := sb.NewSector()
	require.NoError(err)

	d1Data := []byte("hello world")
	d1 := &PieceInfo{
		DealID: 5,
		Size:   uint64(len(d1Data)),
	}

	if err := sector.WritePiece(d1, bytes.NewReader(d1Data)); err != nil {
		t.Fatal(err)
	}

	ag := types.NewAddressForTestGetter()
	ss, err := sb.Seal(sector, ag(), filecoinParameters)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ss.merkleRoot)
	t.Log(ss.replicaData)
}

func requireSectorBuilder(require *require.Assertions, nd *Node, sectorSize int) *SectorBuilder {
	smc, err := NewSectorBuilder(nd, sectorSize, sectorDirsForTest)
	require.NoError(err)
	return smc
}

func requirePieceInfo(require *require.Assertions, nd *Node, bytes []byte) *PieceInfo {
	data := dag.NewRawNode(bytes)
	err := nd.Blockservice.AddBlock(data)
	require.NoError(err)
	return &PieceInfo{
		Ref:    data.Cid(),
		Size:   uint64(len(bytes)),
		DealID: 0, // FIXME parameterize
	}
}

func TestSectorBuilder(t *testing.T) {
	defer sectorDirsForTest.CleanupSectorDirs()
	assert := assert.New(t)
	require := require.New(t)
	ctx := context.Background()

	fname := newSectorLabel()
	assert.Len(fname, 32) // Sanity check, nothing more.

	nd := MakeOfflineNode(t)

	sb := requireSectorBuilder(require, nd, 60)

	assertMetadataMatch := func(sector *Sector, pieces int) {
		meta := sector.SectorMetadata()
		assert.Len(meta.Pieces, pieces)

		// persisted and calculated metadata match.
		metaPersisted, err := sb.GetMeta(sector.Label)
		assert.NoError(err)
		assert.Equal(metaPersisted, meta)

		sealed := sector.sealed
		if sealed != nil {
			sealedMeta := sealed.SealedSectorMetadata()
			sealedMetaPersisted, err := sb.GetSealedMeta(sealed.merkleRoot)
			assert.NoError(err)
			assert.Equal(sealedMeta, sealedMetaPersisted)
		}
	}

	requireAddPiece := func(s string) {
		err := sb.AddPiece(ctx, requirePieceInfo(require, nd, []byte(s)))
		assert.NoError(err)

	}

	assertMetadataMatch(sb.CurSector, 0)

	// New paths are in the right places.
	stagingPath, _ := sb.newSectorPath()
	sealedPath, _ := sb.newSealedSectorPath()
	assert.Contains(stagingPath, sb.stagingDir)
	assert.Contains(sealedPath, sb.sealedDir)

	// New paths are generated each time.
	stpath2, _ := sb.newSectorPath()
	sepath2, _ := sb.newSealedSectorPath()
	assert.NotEqual(stagingPath, stpath2)
	assert.NotEqual(sealedPath, sepath2)

	sector := sb.CurSector
	assert.NotNil(sector.file)
	assert.IsType(&os.File{}, sector.file)

	assertMetadataMatch(sb.CurSector, 0)
	text := "What's our vector, sector?" // len(text) = 26
	requireAddPiece(text)
	assert.Equal(sector, sb.CurSector)
	all := text

	assertMetadataMatch(sector, 1)

	d := requireReadAll(require, sector)
	assert.Equal(all, string(d))
	assert.Nil(sector.sealed)

	text2 := "We have clearance, Clarence." // len(text2) = 28
	requireAddPiece(text2)
	assert.Equal(sector, sb.CurSector)
	all += text2

	d2 := requireReadAll(require, sector)
	assert.Equal(all, string(d2))
	assert.Nil(sector.sealed)

	assert.NotContains(string(sector.data), string(d2)) // Document behavior: data only set at sealing time.

	// persisted and calculated metadata match.
	assertMetadataMatch(sector, 2)

	text3 := "I'm too sexy for this sector." // len(text3) = 29
	requireAddPiece(text3)
	time.Sleep(100 * time.Millisecond) // Wait for sealing to finish: FIXME, don't sleep.
	assert.NotEqual(sector, sb.CurSector)

	d3 := requireReadAll(require, sector)
	assert.Equal(all, string(d3)) // Initial sector still contains initial data.

	assert.Contains(string(sector.data), string(d3)) // Sector data has been set. 'Contains' because it is padded.

	// persisted and calculated metadata match after a sector is sealed.
	assertMetadataMatch(sector, 2)

	newSector := sb.CurSector
	d4 := requireReadAll(require, newSector)
	assertMetadataMatch(newSector, 1)

	assert.Equal(text3, d4)
	sealed := sector.sealed
	assert.NotNil(sealed)
	assert.Nil(newSector.sealed)

	assert.Equal(sealed.baseSector, sector)
	sealedData, err := sealed.ReadFile()
	assert.NoError(err)
	assert.Equal(sealed.replicaData, sealedData)

	meta := sb.CurSector.SectorMetadata()
	assert.Len(meta.Pieces, 1)
	assert.Equal(uint64(60), meta.Size)
	assert.Equal(60-len(text3), int(meta.Free))

	text4 := "I am text, and I am long. My reach exceeds my grasp exceeds exceeds my allotted space."
	err = sb.AddPiece(ctx, requirePieceInfo(require, nd, []byte(text4)))
	assert.EqualError(err, ErrPieceTooLarge.Error())
}

func TestSectorBuilderMetadata(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	//ctx := context.Background()

	fname := newSectorLabel()
	assert.Len(fname, 32) // Sanity check, nothing more.

	nd := MakeOfflineNode(t)

	sb := requireSectorBuilder(require, nd, 60)

	label := "SECTORFILENAMEWHATEVER"

	k := sb.metadataKey(label).String()
	// Don't accidentally test Datastore namespacing implementation.
	assert.Contains(k, "sectors")
	assert.Contains(k, "metadata")
	assert.Contains(k, label)

	merkleRoot := ([]byte)("someMerkleRootLOL")
	k2 := sb.sealedMetadataKey(merkleRoot).String()
	// Don't accidentally test Datastore namespacing implementation.
	assert.Contains(k2, "sealedSectors")
	assert.Contains(k2, "metadata")
	assert.Contains(k2, merkleString(merkleRoot))
}

func requireReadAll(require *require.Assertions, sector *Sector) string {
	data, err := sector.ReadFile()
	require.NoError(err)

	return string(data)
}