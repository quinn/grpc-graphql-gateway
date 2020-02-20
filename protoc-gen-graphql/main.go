package main

import (
	"log"
	"os"

	"io/ioutil"

	"github.com/golang/protobuf/proto"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/ysugimoto/grpc-graphql-gateway/protoc-gen-graphql/generator"
	"github.com/ysugimoto/grpc-graphql-gateway/protoc-gen-graphql/spec"
)

func main() {
	var genError error

	resp := &plugin.CodeGeneratorResponse{}
	defer func() {
		// If some error has been occurred in generate process,
		// add error message to plugin response
		if genError != nil {
			message := genError.Error()
			resp.Error = &message
		}
		buf, err := proto.Marshal(resp)
		if err != nil {
			log.Fatalln(err)
		}
		os.Stdout.Write(buf)
	}()

	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		genError = err
		return
	}

	var req plugin.CodeGeneratorRequest
	if err := proto.Unmarshal(buf, &req); err != nil {
		genError = err
		return
	}

	log.Println(req.GetFileToGenerate())
	for _, f := range req.GetProtoFile() {
		log.Println(f.GetName())
	}

	var args *spec.Params
	if req.Parameter != nil {
		args, err = spec.NewParams(req.GetParameter())
		if err != nil {
			genError = err
			return
		}
	}

	// We're dealing with each descriptors to out wrapper struct
	// in order to access easily plugin options, package name, comment, etc...
	var files []*spec.File
	for _, f := range req.GetProtoFile() {
		files = append(files, spec.NewFile(f))
	}

	g := generator.New(generator.GenerationTypeGo, files, args)
	genFiles, err := g.Generate(goTemplate, req.GetFileToGenerate())
	if err != nil {
		genError = err
		return
	}
	resp.File = append(resp.File, genFiles...)
}

var goTemplate = `
// Code generated by proroc-gen-graphql, DO NOT EDIT.
package {{ .RootPackage.Name }}

import (
	"encoding/json"

	"github.com/graphql-go/graphql"
{{- if .Queries }}{{ if .Mutations }}
	"github.com/ysugimoto/grpc-graphql-gateway/runtime"
	"google.golang.org/grpc"
{{- end }}{{ end }}

{{- range .Packages }}
	{{ .Name }} "{{ .Path }}"
{{ end }}
)

var _ = json.Marshal
var _ = json.Unmarshal

{{ range .Types -}}
var Gql__type_{{ .TypeName }} = graphql.NewObject(graphql.ObjectConfig{
	Name: "{{ .TypeName }}",
	{{- if .Comment }}
	Description: "{{ .Comment }}",
	{{- end }}
	Fields: graphql.Fields {
{{- range .Fields }}
		"{{ .Name }}": &graphql.Field{
			Type: {{ .FieldType $.RootPackage.Path }},
			{{- if .Comment }}
			Description: "{{ .Comment }}",
			{{- end }}
		},
{{- end }}
	},
}) // message {{ .Name }} in {{ .Filename }}

{{ end }}

{{ range .Enums -}}
var Gql__enum_{{ .Name }} = graphql.NewEnum(graphql.EnumConfig{
	Name: "{{ .Name }}",
	Values: graphql.EnumValueConfigMap{
{{- range .Values }}
		"{{ .Name }}": &graphql.EnumValueConfig{
			{{- if .Comment }}
			Description: "{{ .Comment }}",
			{{- end }}
			Value: {{ .Number }},
		},
{{- end }}
	},
}) // enum {{ .Name }} in {{ .Filename }}
{{ end }}

{{ range .Inputs -}}
var Gql__input_{{ .TypeName }} = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "{{ .TypeName }}",
	Fields: graphql.InputObjectConfigFieldMap{
{{- range .Fields }}
		"{{ .Name }}": &graphql.InputObjectFieldConfig{
			{{- if .Comment }}
			Description: "{{ .Comment }}",
			{{- end }}
			Type: {{ .FieldTypeInput $.RootPackage.Path }},
		},
{{- end }}
	},
}) // message {{ .Name }} in {{ .Filename }}

{{ end }}

{{- if .Queries }}{{ if .Mutations }}
// graphql__resolver_{{ .RootPackage.CamelName }} is a struct for making query, mutation and resolve fields.
// This struct must be implemented runtime.SchemaBuilder interface.
type graphql__resolver_{{ .RootPackage.CamelName }} struct {
	// grpc client connection.
	// this connection may be provided by user, then isAutoConnection should be false
	conn *grpc.ClientConn

	// isAutoConnection indicates that the grpc connection is opened by this handler.
	// If true, this handler opens connection automatically, and it should be closed on Close() method.
	isAutoConnection bool
}

// Close() closes grpc connection if it is opened automatically.
func (x *graphql__resolver_{{ .RootPackage.CamelName }}) Close() error {
	// nothing to do because the connection is supplied by user, and it should be closed user themselves.
	if !x.isAutoConnection {
		return nil
	}
	return x.conn.Close()
}

// GetQueries returns acceptable graphql.Fields for Query.
func (x *graphql__resolver_{{ .RootPackage.CamelName }}) GetQueries() graphql.Fields {
	return graphql.Fields{
{{- range .Queries }}
		"{{ .QueryName }}": &graphql.Field{
			Type: {{ .QueryType }},
			{{- if .Comment }}
			Description: "{{ .Comment }}",
			{{- end }}
			{{- if .Args }}
			Args: graphql.FieldConfigArgument{
			{{- range .Args }}
				"{{ .Name }}": &graphql.ArgumentConfig{
					Type: {{ .FieldType $.RootPackage.Path }},
					{{- if .Comment }}
					Description: "{{ .Comment }}",
					{{- end }}
					{{- if .DefaultValue }}
					DefaultValue: {{ .DefaultValue }},
					{{- end }}
				},
			{{- end }}
			},
			{{- end }}
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				var req *{{ .RequestType }}
				if err := runtime.MarshalRequest(p.Args, req); err != nil {
					return nil, err
				}
				client := {{ .Package }}New{{ .Service.Name }}Client(x.conn)
				resp, err := client.{{ .Method.Name }}(p.Context, req)
				if err != nil {
					return nil, err
				}
				{{- if .Expose }}
				return resp.Get{{ .Expose }}(), nil
				{{- else }}
				return resp, nil
				{{- end }}
			},
		},
{{- end }}
	}
}

// GetMutations returns acceptable graphql.Fields for Mutation.
func (x *graphql__resolver_{{ .RootPackage.CamelName }}) GetMutations() graphql.Fields {
	return graphql.Fields{
{{- range .Mutations }}
		"{{ .MutationName }}": &graphql.Field{
			Type: {{ .MutationType }},
			{{- if .Comment }}
			Description: "{{ .Comment }}",
			{{ end }}
			Args: graphql.FieldConfigArgument{
				"{{ .InputName }}": &graphql.ArgumentConfig{
					Type: Gql__input_{{ .Input.Name }},
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				var req *{{ .RequestType }}
				if err := runtime.MarshalRequest(p.Args, req); err != nil {
					return nil, err
				}
				client := {{ .Package }}New{{ .Service.Name }}Client(x.conn)
				resp, err := client.{{ .Method.Name }}(p.Context, req)
				if err != nil {
					return nil, err
				}
				{{- if .Expose }}
				return resp.Get{{ .Expose }}(), nil
				{{- else }}
				return resp, nil
				{{- end }}
			},
		},
{{ end }}
	}
}

// Register package divided graphql handler "without" *grpc.ClientConn,
// therefore gRPC connection will be opened and closed automatically.
// Occasionally you may worry about open/close performance for each handling graphql request,
// then you can call Register{{ .RootPackage.CamelName }}GraphqlHandler with *grpc.ClientConn manually.
func Register{{ .RootPackage.CamelName }}Graphql(mux *runtime.ServeMux) error {
	return Register{{ .RootPackage.CamelName }}GraphqlHandler(mux, nil)
}

// Register package divided graphql handler "with" *grpc.ClientConn.
// this function accepts your defined grpc connection, so that we reuse that and never close connection inside.
// You need to close it maunally when application will terminate.
// Otherwise, the resolver opens connection automatically and then you need to define host with FileOption like:
//
// syntax = "proto3";
// package example;
//
// option (graphql.service) = {
//   host: "localhost:50051";
//   insecure: true or false;
// };
//
// ...some definitions
//
func Register{{ .RootPackage.CamelName }}GraphqlHandler(mux *runtime.ServeMux, conn *grpc.ClientConn) (err error) {
	var isAutoConnection bool
	if conn == nil {
		isAutoConnection = true
		conn, err = grpc.Dial("localhost:50051", grpc.WithInsecure())
		if err != nil {
			return
		}
	}
	mux.AddHandler(&graphql__resolver_{{ .RootPackage.CamelName }}{conn, isAutoConnection})
	return
}
{{ end }}{{ end }}
`
