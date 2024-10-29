{{/*
Return the proper istio operator image name
*/}}
{{- define "operator.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.image "global" .Values.global "defaultTag" .Chart.Version) }}
{{- end -}}
