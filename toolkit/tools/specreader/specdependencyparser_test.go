// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplePackage(t *testing.T) {
	testValid(t, "a", &DependencyExpr{
		Type: PackageExprType,
		Package: &PackageExpr{
			Name: "a",
		},
	})
}

func TestVersionCompare(t *testing.T) {
	testValid(t, "a >= 1", &DependencyExpr{
		Type: PackageExprType,
		Package: &PackageExpr{
			Name:              "a",
			VersionComparison: ">=",
			Version:           "1",
		},
	})
}

func TestSimpleOr(t *testing.T) {
	testValid(t, "(a or b)", &DependencyExpr{
		Type: OrExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func TestChainedOr(t *testing.T) {
	testValid(t, "(a or b or c)", &DependencyExpr{
		Type: OrExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "c",
				},
			},
		},
	})
}

func TestChainedOrWithVersions(t *testing.T) {
	testValid(t, "(a = 1 or b >= 2 or c <= 3)", &DependencyExpr{
		Type: OrExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "a",
					VersionComparison: "=",
					Version:           "1",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "b",
					VersionComparison: ">=",
					Version:           "2",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "c",
					VersionComparison: "<=",
					Version:           "3",
				},
			},
		},
	})
}

func TestSimpleAnd(t *testing.T) {
	testValid(t, "(a and b)", &DependencyExpr{
		Type: AndExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func TestChainedAnd(t *testing.T) {
	testValid(t, "(a and b and c)", &DependencyExpr{
		Type: AndExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "c",
				},
			},
		},
	})
}

func TestChainedAndWithVersions(t *testing.T) {
	testValid(t, "(a = 1 and b >= 2 and c <= 3)", &DependencyExpr{
		Type: AndExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "a",
					VersionComparison: "=",
					Version:           "1",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "b",
					VersionComparison: ">=",
					Version:           "2",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name:              "c",
					VersionComparison: "<=",
					Version:           "3",
				},
			},
		},
	})
}

func TestNested(t *testing.T) {
	testValid(t, "(a and (b or c))", &DependencyExpr{
		Type: AndExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: OrExprType,
				Clauses: []*DependencyExpr{
					{
						Type: PackageExprType,
						Package: &PackageExpr{
							Name: "b",
						},
					},
					{
						Type: PackageExprType,
						Package: &PackageExpr{
							Name: "c",
						},
					},
				},
			},
		},
	})
}

func TestNestedReverse(t *testing.T) {
	testValid(t, "((a or b) and c)", &DependencyExpr{
		Type: AndExprType,
		Clauses: []*DependencyExpr{
			{
				Type: OrExprType,
				Clauses: []*DependencyExpr{
					{
						Type: PackageExprType,
						Package: &PackageExpr{
							Name: "a",
						},
					},
					{
						Type: PackageExprType,
						Package: &PackageExpr{
							Name: "b",
						},
					},
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "c",
				},
			},
		},
	})
}

func TestIf(t *testing.T) {
	testValid(t, "(a if b)", &DependencyExpr{
		Type: IfExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func TestIfElse(t *testing.T) {
	testValid(t, "(a if b else c)", &DependencyExpr{
		Type: IfExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "c",
				},
			},
		},
	})
}

func TestUnless(t *testing.T) {
	testValid(t, "(a unless b)", &DependencyExpr{
		Type: UnlessExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func TestUnlessElse(t *testing.T) {
	testValid(t, "(a unless b else c)", &DependencyExpr{
		Type: UnlessExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "c",
				},
			},
		},
	})
}

func TestWith(t *testing.T) {
	testValid(t, "(a with b)", &DependencyExpr{
		Type: WithExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func TestWithout(t *testing.T) {
	testValid(t, "(a without b)", &DependencyExpr{
		Type: WithoutExprType,
		Clauses: []*DependencyExpr{
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "a",
				},
			},
			{
				Type: PackageExprType,
				Package: &PackageExpr{
					Name: "b",
				},
			},
		},
	})
}

func testValid(t *testing.T, stringValue string, expectedValue *DependencyExpr) {
	value, err := ParseDependencyExpr(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, expectedValue, value)
}
