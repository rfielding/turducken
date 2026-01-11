package store

import "sync"

type Doc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Mime  string `json:"mime"`
	// Content omitted from List() responses; fetched separately
	Content string `json:"-"`
}

type DocStore struct {
	mu   sync.RWMutex
	meta map[string]Doc
	body map[string]string
}

func NewDocStore() *DocStore {
	return &DocStore{
		meta: map[string]Doc{},
		body: map[string]string{},
	}
}

func (s *DocStore) Put(id, title, mime, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta[id] = Doc{ID: id, Title: title, Mime: mime}
	s.body[id] = content
}

func (s *DocStore) Get(id string) (Doc, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.meta[id]
	if !ok {
		return Doc{}, false
	}
	d.Content = s.body[id]
	return d, true
}

func (s *DocStore) List() []Doc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Doc, 0, len(s.meta))
	for _, d := range s.meta {
		out = append(out, d)
	}
	return out
}

func (s *DocStore) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]string{}
	for id := range s.meta {
		out[id] = s.body[id]
	}
	return out
}
