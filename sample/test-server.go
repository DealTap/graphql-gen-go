package main

import (
  "strconv"
  "strings"

  "github.com/dealtap/graphql-gen-go/sample/api"
)

const Port = "7050"
const UniqueIdBase = 1000

var people = []*api.Person{
  {
    ID:    "1000",
    Name:  "Luke Skywalker",
    Email: "luke.skywalker@star.wars",
  },
  {
    ID:    "1001",
    Name:  "Darth Vader",
    Email: "darth.vader@star.wars",
  },
  {
    ID:    "1002",
    Name:  "Han Solo",
    Email: "han.solo@star.wars",
  },
  {
    ID:    "1003",
    Name:  "Leia Organa",
    Email: "leia.organa@star.wars",
  },
}

var folders = []*api.Folder{
  {
    ID:   "1000",
    Name: "Test 1",
  },
  {
    ID:   "1001",
    Name: "Test 2",
  },
}

var files = []*api.File{
  {
    ID:     "1000",
    Name:   "Test file 1 in folder 1",
    Folder: *folders[0],
  },
  {
    ID:     "1001",
    Name:   "Test file 2 in folder 1",
    Folder: *folders[0],
  },
  {
    ID:     "1002",
    Name:   "Test file 3 in folder 2",
    Folder: *folders[1],
  },
  {
    ID:     "1003",
    Name:   "Test file 4 in folder 2",
    Folder: *folders[1],
  },
}

type resolver struct{}

func (r *resolver) Person(request api.PersonRequest) api.PersonResolver {
  var p *api.Person
  for _, prs := range people {
    if prs.ID == request.ID {
      p = prs
    }
  }
  return api.PersonResolver{p}
}

func (r *resolver) CreateFile(request api.CreateFileRequest) api.FileResolver {

  var f *api.File

  // create a file ONLY when folder id is provided and folder exists
  if len(request.FolderId) > 0 {
    for _, fl := range folders {
      if fl.ID == request.FolderId {
        id := UniqueIdBase + len(files)
        f = &api.File{
          ID:     strconv.Itoa(id),
          Name:   request.File.Name,
          Folder: *fl,
        }

        files = append(files, f)
        fl.Files = append(fl.Files, f)

        break
      }
    }
  }

  return api.FileResolver{f}
}

func (r *resolver) CreateFolder(request api.CreateFolderRequest) api.FolderResolver {
  id := UniqueIdBase + len(folders)
  f := &api.Folder{
    ID:   strconv.Itoa(id),
    Name: request.Folder.Name,
  }
  folders = append(folders, f)
  return api.FolderResolver{f}
}

func (r *resolver) CreatePerson(request api.CreatePersonRequest) api.PersonResolver {
  id := UniqueIdBase + len(people)
  p := &api.Person{
    ID:    strconv.Itoa(id),
    Name:  request.Person.Name,
    Email: request.Person.Email,
  }
  people = append(people, p)
  return api.PersonResolver{p}
}

func (r *resolver) Search(request api.SearchRequest) []*api.SearchResultResolver {

  var result []*api.SearchResultResolver

  for _, f := range files {
    if strings.Contains(f.Name, request.Text) {
      result = append(result, &api.SearchResultResolver{&api.FileResolver{f}})
    }
  }

  for _, f := range folders {
    if strings.Contains(f.Name, request.Text) {
      result = append(result, &api.SearchResultResolver{&api.FolderResolver{f}})
    }
  }

  return result
}

func main() {

  // a hacky way to update folders to reference files
  folders[0].Files = []*api.File{
    files[0],
    files[1],
  }
  folders[1].Files = []*api.File{
    files[2],
    files[3],
  }

  // set friends
  people[0].Friends = []*api.Person{
    people[2],
    people[3],
  }

  gqlSrv := api.NewGqlServer(&resolver{}, Port, nil)
  err := gqlSrv.Serve()
  if err != nil {
    panic(err)
  }
}
