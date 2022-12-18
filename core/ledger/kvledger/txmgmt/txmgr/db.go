package txmgr

import "github.com/osdi23p228/fabric/core/ledger"

// strawman codes vvvvvvvvvvvvvvvvvvvvvvv
type VisibleDB struct {
	sessions map[string]*SessionDB
}

func NewVisibleDB() *VisibleDB {
	vdb := &VisibleDB{
		sessions: make(map[string]*SessionDB),
	}
	return vdb
}

func (vdb *VisibleDB) Get(key string, session string) *ledger.VersionedValue {
	sdb, exist := vdb.sessions[key]
	if !exist {
		return nil
	}
	return sdb.Get(key)
}

func (vdb *VisibleDB) Set(key string, value *ledger.VersionedValue, session string) {
	sdb, exist := vdb.sessions[key]
	if !exist {
		sdb = NewSessionDB()
		vdb.sessions[key] = sdb
	}
	sdb.Set(key, value)
}

type SessionDB struct {
	data map[string]*ledger.VersionedValue
}

func NewSessionDB() *SessionDB {
	sdb := &SessionDB{
		data: make(map[string]*ledger.VersionedValue),
	}
	return sdb
}

func (sdb *SessionDB) Get(key string) *ledger.VersionedValue {
	vv, exist := sdb.data[key]
	if !exist {
		return nil
	}
	return vv
}

func (sdb *SessionDB) Set(key string, value *ledger.VersionedValue) {
	sdb.data[key] = value
}

// strawman codes ^^^^^^^^^^^^^^^^^^^^^^^
