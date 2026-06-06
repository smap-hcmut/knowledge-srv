package qdrant

import (
	"strconv"

	pb "github.com/qdrant/go-client/qdrant"
)

// PointIDString returns a stable string form for Qdrant point IDs (UUID or numeric hash id).
func PointIDString(id *pb.PointId) string {
	if id == nil {
		return ""
	}
	if u := id.GetUuid(); u != "" {
		return u
	}
	return strconv.FormatUint(id.GetNum(), 10)
}

// valueToInterface converts a qdrant Value to a Go interface{} (for payload extraction).
func valueToInterface(v *pb.Value) interface{} {
	if v == nil {
		return nil
	}
	switch v.GetKind().(type) {
	case *pb.Value_NullValue:
		return nil
	case *pb.Value_DoubleValue:
		return v.GetDoubleValue()
	case *pb.Value_IntegerValue:
		return v.GetIntegerValue()
	case *pb.Value_StringValue:
		return v.GetStringValue()
	case *pb.Value_BoolValue:
		return v.GetBoolValue()
	case *pb.Value_StructValue:
		st := v.GetStructValue()
		if st == nil {
			return nil
		}
		m := make(map[string]interface{})
		for key, val := range st.GetFields() {
			m[key] = valueToInterface(val)
		}
		return m
	case *pb.Value_ListValue:
		list := v.GetListValue()
		if list == nil {
			return nil
		}
		vals := list.GetValues()
		s := make([]interface{}, len(vals))
		for i, val := range vals {
			s[i] = valueToInterface(val)
		}
		return s
	default:
		return nil
	}
}
