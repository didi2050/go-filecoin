package commands_test

import (
	"fmt"
	"github.com/filecoin-project/go-filecoin/fixtures"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
	"gx/ipfs/QmPVkJMTeRC6iBByPWdrRkD3BE5UXsj5HPzb4kPqL186mS/testify/require"
)

func requireSchemaConformance(t *testing.T, jsonBytes []byte, schemaName string) { // nolint: deadcode
	wdir, _ := os.Getwd()
	rLoader := gojsonschema.NewReferenceLoader(fmt.Sprintf("file://%s/schema/%s.schema.json", wdir, schemaName))
	jLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(rLoader, jLoader)
	require.NoError(t, err)

	for _, desc := range result.Errors() {
		t.Errorf("- %s\n", desc)
	}

	require.Truef(t, result.Valid(), "Error schema validating: %s", string(jsonBytes))
}

func makeTestDaemonWithMinerAndStart(t *testing.T) *th.TestDaemon {
	daemon := th.NewDaemon(
		t,
		th.DefaultAddress(fixtures.TestAddresses[0]),
		th.WithMiner(fixtures.TestMiners[0]),
		th.KeyFile(fixtures.KeyFilePaths()[0]),
	).Start()
	return daemon
}
