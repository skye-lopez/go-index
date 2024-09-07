# go-index
A more useable go index

### CLI

#### update-db
`go-index update-db`

Upserts all index data to local leveldb(s). Keeps a `lastwrite` time in `.go-index/store` to ensure quick updates.

##### .go-index/store
A simple KV store that contains a list of all unique packages in the index.
`[]byte{PackagePath}=>[]byte{PackagePath}`

##### .go-index/search 
A prefix map to provide the ability of fast searching.
```
[]byte{Prefix}=>[]byte{DeliminitedPackages}

Value string struct:
{PackagePath}~{PackagePath}~{PackagePath} ... 

