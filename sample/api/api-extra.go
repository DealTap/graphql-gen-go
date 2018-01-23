package api

/**
 * Manual implementation of resolver functions that require
 * handling of complicated logic/filtering
 */

func (r PersonResolver) Friends(args FriendsRequest) []*PersonResolver {

  from := 0
  // In a real app, this not a good way to find `from`
  if args.After != nil {
    for i, f := range r.R.Friends {
      if *args.After == f.ID {
        from = i + 1
        break
      }
    }
  }

  to := len(r.R.Friends)
  if args.First != nil {
    to = from + int(*args.First)
    if to > len(r.R.Friends) {
      to = len(r.R.Friends)
    }
  }

  friends := make([]*PersonResolver, to-from)
  for i := range friends {
    friends[i] = &PersonResolver{r.R.Friends[from+i]}
  }

  return friends
}
