package store

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

type Post struct {
	ID           string    `json:"id"`
	AuthorPeerID string    `json:"authorPeerId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	Attachments  []string  `json:"attachments,omitempty"`
	Signature    string    `json:"signature"`
}

type Profile struct {
	PeerID      string   `json:"peerId"`
	DisplayName string   `json:"displayName"`
	Bio         string   `json:"bio"`
	AvatarHash  string   `json:"avatarHash,omitempty"`
	Addresses   []string `json:"addresses,omitempty"`
}

type Store struct {
	db        *badger.DB
	localPeer string
}

func New(dataDir string, localPeer string) (*Store, error) {
	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, localPeer: localPeer}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SavePost(post *Post) error {
	if post.ID == "" {
		post.ID = uuid.New().String()
	}
	if post.CreatedAt.IsZero() {
		post.CreatedAt = time.Now()
	}
	post.AuthorPeerID = s.localPeer

	data, err := json.Marshal(post)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set([]byte("post:local:"+post.ID), data); err != nil {
			return err
		}
		return txn.Set([]byte("post:all:"+post.ID), data)
	})
}

func (s *Store) GetPost(id string) (*Post, error) {
	var post Post
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("post:all:" + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &post)
		})
	})
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *Store) GetLocalPosts(since time.Time) ([]Post, error) {
	var posts []Post
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("post:local:")
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var post Post
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &post)
			})
			if err != nil {
				return err
			}
			if post.CreatedAt.After(since) {
				posts = append(posts, post)
			}
		}
		return nil
	})
	return posts, err
}

func (s *Store) GetAllPosts() ([]Post, error) {
	var posts []Post
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("post:all:")
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var post Post
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &post)
			})
			if err != nil {
				return err
			}
			posts = append(posts, post)
		}
		return nil
	})
	return posts, err
}

func (s *Store) SaveRemotePost(post *Post) error {
	data, err := json.Marshal(post)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("post:all:"+post.ID), data)
	})
}

func (s *Store) UpdatePostSignature(id, signature string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("post:all:" + id))
		if err != nil {
			return err
		}
		var post Post
		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &post)
		})
		if err != nil {
			return err
		}
		post.Signature = signature
		data, err := json.Marshal(post)
		if err != nil {
			return err
		}
		if err := txn.Set([]byte("post:local:"+id), data); err != nil {
			return err
		}
		return txn.Set([]byte("post:all:"+id), data)
	})
}

func (s *Store) GetProfile() (*Profile, error) {
	var profile Profile
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("profile:local"))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &profile)
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return &Profile{PeerID: s.localPeer}, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (s *Store) SaveProfile(profile *Profile) error {
	profile.PeerID = s.localPeer
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("profile:local"), data)
	})
}

func (s *Store) SaveRemoteProfile(profile *Profile) error {
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("profile:remote:"+profile.PeerID), data)
	})
}

func (s *Store) SaveRemoteProfileMerge(profile *Profile) error {
	return s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("profile:remote:" + profile.PeerID))
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}

		var existingProfile Profile
		if err == nil {
			err = item.Value(func(val []byte) error {
				return json.Unmarshal(val, &existingProfile)
			})
			if err != nil {
				return err
			}
			if len(existingProfile.Addresses) > 0 {
				profile.Addresses = existingProfile.Addresses
			}
		}

		data, err := json.Marshal(profile)
		if err != nil {
			return err
		}
		return txn.Set([]byte("profile:remote:"+profile.PeerID), data)
	})
}

func (s *Store) GetRemoteProfile(peerID string) (*Profile, error) {
	var profile Profile
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("profile:remote:" + peerID))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &profile)
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return &Profile{PeerID: peerID}, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (s *Store) GetKnownPeers() []string {
	var peers []string
	seen := make(map[string]bool)
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("profile:remote:")
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())
			peerID := key[len("profile:remote:"):]
			if !seen[peerID] {
				peers = append(peers, peerID)
				seen[peerID] = true
			}
		}
		return nil
	})
	return peers
}

func (s *Store) GetKnownPeersWithProfiles() []*Profile {
	var profiles []*Profile
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("profile:remote:")
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var profile Profile
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &profile)
			})
			if err != nil {
				continue
			}
			profiles = append(profiles, &profile)
		}
		return nil
	})
	return profiles
}
