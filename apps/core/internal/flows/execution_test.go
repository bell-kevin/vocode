package flows

import "testing"

func TestRouteExecution_coversEverySpecRoute(t *testing.T) {
	for _, fid := range []ID{Root, Select, SelectFile} {
		spec := SpecFor(fid)
		for _, r := range spec.Routes {
			pol := RouteExecution(fid, r.ID)
			switch pol {
			case ExecutionImmediate, ExecutionSerialized:
			default:
				t.Fatalf("unknown execution policy %v for flow %q route %q", pol, fid, r.ID)
			}
		}
	}
}

func TestRouteExecution_unknownRouteIsSerialized(t *testing.T) {
	if RouteExecution(Root, "totally_unknown_route_xyz") != ExecutionSerialized {
		t.Fatal("expected unknown route to default to serialized")
	}
}
