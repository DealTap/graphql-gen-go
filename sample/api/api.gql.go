package api

import (
  "github.com/graph-gophers/graphql-go"
)

type File struct {
  ID     string
  Name   string
  Folder Folder
}

type FileResolver struct {
  R *File
}

func (r FileResolver) ID() graphql.ID {
  id := graphql.ID(r.R.ID)
  return id
}

func (r FileResolver) Name() string {
  return r.R.Name
}

func (r FileResolver) Folder() FolderResolver {
  return FolderResolver{&r.R.Folder}
}

type FileInput struct {
  Name string
}

type Folder struct {
  ID    string
  Name  string
  Files []*File
}

type FolderResolver struct {
  R *Folder
}

func (r FolderResolver) ID() graphql.ID {
  id := graphql.ID(r.R.ID)
  return id
}

func (r FolderResolver) Name() string {
  return r.R.Name
}

func (r FolderResolver) Files() []*FileResolver {
  items := []*FileResolver{}
  for _, itm := range r.R.Files {
    items = append(items, &FileResolver{itm})
  }
  return items
}

type FolderInput struct {
  ID   *string
  Name string
}

type Person struct {
  Email   string
  Friends []*Person
  ID      string
  Name    string
}

type PersonResolver struct {
  R *Person
}

func (r PersonResolver) ID() graphql.ID {
  id := graphql.ID(r.R.ID)
  return id
}

func (r PersonResolver) Name() string {
  return r.R.Name
}

func (r PersonResolver) Email() string {
  return r.R.Email
}

type PersonInput struct {
  Name  string
  Email string
}

type SearchResultResolver struct {
  Result interface{}
}

func (r *SearchResultResolver) ToFolder() (*FolderResolver, bool) {
  res, ok := r.Result.(*FolderResolver)
  return res, ok
}

func (r *SearchResultResolver) ToFile() (*FileResolver, bool) {
  res, ok := r.Result.(*FileResolver)
  return res, ok
}

type CreateFileRequest struct {
  FolderId string
  File     FileInput
}

type CreatePersonRequest struct {
  Person PersonInput
}

type CreateFolderRequest struct {
  Folder FolderInput
}

type FriendsRequest struct {
  First *int32
  After *string
}

type SearchRequest struct {
  Text string
}

type PersonRequest struct {
  ID string
}

type GqlResolver interface {
  CreatePerson(CreatePersonRequest) PersonResolver
  CreateFolder(CreateFolderRequest) FolderResolver
  CreateFile(CreateFileRequest) FileResolver
  Person(PersonRequest) PersonResolver
  Search(SearchRequest) []*SearchResultResolver
}

var Schema = `

schema {
  query: Query
  mutation: Mutation
}

type Query {
  person(id: ID!): Person!
  search(text: String!): [SearchResult]!
}

type Mutation {
  createPerson(person: PersonInput!): Person!
  createFolder(folder: FolderInput!): Folder!
  createFile(folderId: ID!, file: FileInput!): File!
}

input PersonInput {
  name: String!
  email: String!
}

input FolderInput {
  id: ID
  name: String!
}

input FileInput {
  name: String!
}

type Person {
  id: ID!
  name: String!
  email: String!
  friends(first: Int, after: ID): [Person]!
}

type Folder {
  id: ID!
  name: String!
  files: [File]!
}

type File {
  id: ID!
  name: String!
  folder: Folder!
}

union SearchResult = Folder | File

`
