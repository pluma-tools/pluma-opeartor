{{/*
Common Template
*/}}

{{/*
Return the proper image name
Usage:     {{ include "common.images.image" ( dict "imageRoot" .imageRootPath "global" .globalPath "defaultTag" .tagPath) }}
*/}}
{{- define "common.images.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .defaultTag  -}}
{{- if .global }}
    {{- if .global.imageRegistry }}
     {{- $registryName = .global.imageRegistry -}}
    {{- end -}}
{{- end -}}
{{- if .imageRoot.registry }}
    {{- $registryName = .imageRoot.registry  -}}
{{- end -}}
{{- if .imageRoot.tag }}
    {{- $tag = .imageRoot.tag  -}}
{{- end -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else -}}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end -}}
{{- end -}}
