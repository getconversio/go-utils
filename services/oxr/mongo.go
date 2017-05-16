package oxr

import (
	"gopkg.in/mgo.v2"
)

// Mongo Cache is a MongoDB implementation of the OXR cache. It uses an
// existing mgo session to connect to MongoDB rather than setting up it's own.
type MongoCache struct {
	session *mgo.Session
	db      string
	coll    string
}

// Create a new Mongo cache with the given target database and collection
func NewMongoCache(session *mgo.Session, db string, collection string) *MongoCache {
	if collection == "" {
		collection = "oxr"
	}
	return &MongoCache{
		session: session,
		db:      db,
		coll:    collection,
	}
}

// Implements Cache.Add
func (cache *MongoCache) Add(key string, rates Rates) error {
	session := cache.session.Copy()
	defer session.Close()

	// Rates is already ready for Mongo so it can be inserted directly
	return session.DB(cache.db).C(cache.coll).Insert(rates)
}

// Implements Cache.Get
func (cache *MongoCache) Get(key string) (*Rates, error) {
	session := cache.session.Copy()
	defer session.Close()

	rates := new(Rates)
	err := session.DB(cache.db).C(cache.coll).FindId(key).One(rates)
	if err != nil {
		if err == mgo.ErrNotFound {
			// Make sure our custom not found error is returned so the caller
			// can deal with that.
			return nil, NotFoundError{}
		}
		return nil, err
	}

	return rates, nil
}
