// Copyright 2016 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package log

import (
	"context"
	"testing"

	"github.com/ruiaylin/pgparser/utils/tracing"
	"github.com/cockroachdb/logtags"
	opentracing "github.com/opentracing/opentracing-go"
)

func TestAnnotateCtxTags(t *testing.T) {
	ac := AmbientContext{}
	ac.AddLogTag("a", 1)
	ac.AddLogTag("b", 2)

	ctx := ac.AnnotateCtx(context.Background())
	if exp, val := "[a1,b2] test", FormatWithContextTags(ctx, "test"); val != exp {
		t.Errorf("expected '%s', got '%s'", exp, val)
	}

	ctx = context.Background()
	ctx = logtags.AddTag(ctx, "a", 10)
	ctx = logtags.AddTag(ctx, "aa", nil)
	ctx = ac.AnnotateCtx(ctx)

	if exp, val := "[a1,aa,b2] test", FormatWithContextTags(ctx, "test"); val != exp {
		t.Errorf("expected '%s', got '%s'", exp, val)
	}
}


func TestAnnotateCtxNodeStoreReplica(t *testing.T) {
	// Test the scenario of a context being continually re-annotated as it is
	// passed down a call stack.
	n := AmbientContext{}
	n.AddLogTag("n", 1)
	s := n
	s.AddLogTag("s", 2)
	r := s
	r.AddLogTag("r", 3)

	ctx := n.AnnotateCtx(context.Background())
	ctx = s.AnnotateCtx(ctx)
	ctx = r.AnnotateCtx(ctx)
	if exp, val := "[n1,s2,r3] test", FormatWithContextTags(ctx, "test"); val != exp {
		t.Errorf("expected '%s', got '%s'", exp, val)
	}
	if tags := logtags.FromContext(ctx); tags != r.tags {
		t.Errorf("expected %p, got %p", r.tags, tags)
	}
}

func TestResetAndAnnotateCtx(t *testing.T) {
	ac := AmbientContext{}
	ac.AddLogTag("a", 1)

	ctx := context.Background()
	ctx = logtags.AddTag(ctx, "b", 2)
	ctx = ac.ResetAndAnnotateCtx(ctx)
	if exp, val := "[a1] test", FormatWithContextTags(ctx, "test"); val != exp {
		t.Errorf("expected '%s', got '%s'", exp, val)
	}
}
