package sliceutil

func ContainsString(list []string, elem string) bool {
    for _, t := range list {
      if t == elem {
        return true
      }
    }

    return false
}
