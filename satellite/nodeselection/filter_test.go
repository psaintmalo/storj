// Copyright (C) 2023 Storj Labs, Inc.
// See LICENSE for copying information.

package nodeselection

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"storj.io/common/identity/testidentity"
	"storj.io/common/storj"
	"storj.io/common/storj/location"
	"storj.io/common/testcontext"
	"storj.io/common/testrand"
)

func TestCriteria_ExcludeNodeID(t *testing.T) {
	included := testrand.NodeID()
	excluded := testrand.NodeID()

	criteria := NodeFilters{}.WithExcludedIDs([]storj.NodeID{excluded})

	assert.False(t, criteria.Match(&SelectedNode{
		ID: excluded,
	}))

	assert.True(t, criteria.Match(&SelectedNode{
		ID: included,
	}))
}

func TestCriteria_ExcludedNodeNetworks(t *testing.T) {
	criteria := NodeFilters{}
	criteria = append(criteria, ExcludedNodeNetworks{
		&SelectedNode{
			LastNet: "192.168.1.0",
		}, &SelectedNode{
			LastNet: "192.168.2.0",
		},
	})

	assert.False(t, criteria.Match(&SelectedNode{
		LastNet: "192.168.1.0",
	}))

	assert.False(t, criteria.Match(&SelectedNode{
		LastNet: "192.168.2.0",
	}))

	assert.True(t, criteria.Match(&SelectedNode{
		LastNet: "192.168.3.0",
	}))
}

func TestAnnotations(t *testing.T) {
	k := WithAnnotation(NodeFilters{}, "foo", "bar")
	require.Equal(t, "bar", k.GetAnnotation("foo"))

	k = NodeFilters{WithAnnotation(NodeFilters{}, "foo", "bar")}
	require.Equal(t, "bar", k.GetAnnotation("foo"))

	k = Annotation{
		Key:   "foo",
		Value: "bar",
	}
	require.Equal(t, "bar", k.GetAnnotation("foo"))
}

func TestCriteria_Geofencing(t *testing.T) {
	eu := NodeFilters{}.WithCountryFilter(location.NewSet(EuCountries...))
	us := NodeFilters{}.WithCountryFilter(location.NewSet(location.UnitedStates))

	cases := []struct {
		name     string
		country  location.CountryCode
		criteria NodeFilters
		expected bool
	}{
		{
			name:     "US matches US selector",
			country:  location.UnitedStates,
			criteria: us,
			expected: true,
		},
		{
			name:     "Germany is EU",
			country:  location.Germany,
			criteria: eu,
			expected: true,
		},
		{
			name:     "US is not eu",
			country:  location.UnitedStates,
			criteria: eu,
			expected: false,
		},
		{
			name:     "Empty country doesn't match region",
			country:  location.CountryCode(0),
			criteria: eu,
			expected: false,
		},
		{
			name:     "Empty country doesn't match country",
			country:  location.CountryCode(0),
			criteria: us,
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, c.criteria.Match(&SelectedNode{
				CountryCode: c.country,
			}))
		})
	}
}

// BenchmarkNodeFilterFullTable checks performances of rule evaluation on ALL storage nodes.
func BenchmarkNodeFilterFullTable(b *testing.B) {
	filters := NodeFilters{}
	filters = append(filters, NodeFilterFunc(func(node *SelectedNode) bool {
		return true
	}))
	filters = append(filters, NodeFilterFunc(func(node *SelectedNode) bool {
		return true
	}))
	filters = append(filters, NodeFilterFunc(func(node *SelectedNode) bool {
		return true
	}))
	benchmarkFilter(b, filters)
}

func benchmarkFilter(b *testing.B, filters NodeFilters) {
	nodeNo := 25000
	if testing.Short() {
		nodeNo = 20
	}
	nodes := generatedSelectedNodes(b, nodeNo)

	b.ResetTimer()
	c := 0
	for j := 0; j < b.N; j++ {
		for n := 0; n < len(nodes); n++ {
			if filters.Match(nodes[n]) {
				c++
			}
		}
	}

}

func generatedSelectedNodes(b *testing.B, nodeNo int) []*SelectedNode {
	nodes := make([]*SelectedNode, nodeNo)
	ctx := testcontext.New(b)
	for i := 0; i < nodeNo; i++ {
		node := SelectedNode{}
		identity, err := testidentity.NewTestIdentity(ctx)
		require.NoError(b, err)
		node.ID = identity.ID
		node.LastNet = fmt.Sprintf("192.168.%d.0", i%256)
		node.LastIPPort = fmt.Sprintf("192.168.%d.%d:%d", i%256, i%65536, i%1000+1000)
		node.CountryCode = []location.CountryCode{location.None, location.UnitedStates, location.Germany, location.Hungary, location.Austria}[i%5]
		nodes[i] = &node
	}
	return nodes
}
