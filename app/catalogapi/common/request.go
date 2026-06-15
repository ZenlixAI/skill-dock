package common

import platformhttp "skill_registry/internal/platform/http"

var (
	DecodeJSONBody                        = platformhttp.DecodeJSONBody
	DecodeJSONBytes                       = platformhttp.DecodeJSONBytes
	DecodeOptionalReleaseRecordInput      = platformhttp.DecodeOptionalReleaseRecordInput
	DecodeOptionalReleaseRecordInputBytes = platformhttp.DecodeOptionalReleaseRecordInputBytes
)
