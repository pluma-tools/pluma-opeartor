/* eslint-disable */
// @ts-nocheck
/*
* This file is a generated Typescript file for GRPC Gateway, DO NOT MODIFY
*/

import * as GoogleProtobufStruct from "../../google/protobuf/struct.pb"

export enum Phase {
  UNKNOWN = "UNKNOWN",
  RECONCILING = "RECONCILING",
  SUCCEEDED = "SUCCEEDED",
  FAILED = "FAILED",
  DELETING = "DELETING",
}

export type HelmAppSpec = {
  components?: HelmComponent[]
  globalValues?: GoogleProtobufStruct.Struct
  repo?: HelmRepo
}

export type HelmComponent = {
  name?: string
  chart?: string
  version?: string
  componentValues?: GoogleProtobufStruct.Struct
  repo?: HelmRepo
  ignoreGlobalValues?: boolean
}

export type HelmRepo = {
  name?: string
  url?: string
}

export type HelmAppStatus = {
  phase?: Phase
  components?: HelmComponentStatus[]
}

export type HelmComponentStatus = {
  name?: string
  status?: string
  message?: string
  version?: string
  resources?: HelmResourceStatus[]
  resourcesTotal?: number
}

export type HelmResourceStatus = {
  apiVersion?: string
  kind?: string
  name?: string
  namespace?: string
}