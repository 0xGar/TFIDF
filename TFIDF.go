package algorithms

import (
	"document"
	"fmt"
	"logging"
	"math"
	"pool"
	"sort"
	"strings"
)

type Document struct {
	Id               int
	Title            string
	Keywords         []string
}


type Score struct {
	Score float64
	Doc   *Document
}

type DocumentPartial struct {
	Doc             *Document
	tokens          []string //for tf-idf stuff only
	tokensDuplicate []string
	tf              map[string]float64 //this doc's portion of tf for tf-idf
}
type DocumentGroup struct {
	DocPartials []*DocumentPartial
	tfidf       map[int]map[string]float64
	idf         map[string]float64
	isInit      bool
}

func (docGroup *DocumentGroup) Init(docs []*Document, getFromDb bool) {
	var documents []*Document
	if getFromDb {
		documents = fillFromDb()
	} else {
		documents = docs
	}
	for _, doc := range documents {
		docP := &DocumentPartial{
			Doc: doc,
		}
		docGroup.DocPartials = append(docGroup.DocPartials, docP)
	}
	docGroup.makeTFIDF()
	docGroup.isInit = true
}
func (docGroup *DocumentGroup) Exists(id int) bool {
	if !docGroup.isInit {
		return false
	}
	if _, exists := docGroup.tfidf[id]; exists {
		return true
	}
	return false
}
func (docGroup *DocumentGroup) GetRecommendation(id, limit int) ([]*Score, error) {
	if !docGroup.isInit {
		return nil, logging.MakeError("DocumentGroup not initialized.")
	}
	cosSimilarity := docGroup.recommendFromExistingDocumentInTFIDF(id)

	orderedScores := make([]*Score, 0)
	for id, score := range cosSimilarity {
		s := &Score{Score: score, Doc: docGroup.DocPartials[id].Doc}
		orderedScores = append(orderedScores, s)
	}
	sort.Slice(orderedScores, func(i, j int) bool { return orderedScores[i].Score > orderedScores[j].Score })
	return orderedScores[:limit], nil
}

func (docGroup *DocumentGroup) SearchByKeywords(doc *DocumentPartial) (map[int]float64, error) { //document contains search query
	if !docGroup.isInit {
		return nil, logging.MakeError("DocumentGroup not initialized.")
	}
	if len(docGroup.tfidf) < 1 {
		docGroup.makeTFIDF()
	}
	result := make(map[string]float64)
	tf := doc.GetTF(len(docGroup.DocPartials))
	for key := range tf {
		if _, exists := docGroup.idf[key]; exists {
			result[key] = tf[key] * docGroup.idf[key]
		}
	}
	return docGroup.recommendationUnsorted(result), nil
}

func (docGroup *DocumentGroup) recommendFromExistingDocumentInTFIDF(searchId int) map[int]float64 {
	if len(docGroup.tfidf) < 1 {
		docGroup.makeTFIDF()
	}
	return docGroup.recommendationUnsorted(docGroup.tfidf[searchId])
}

func (docGroup *DocumentGroup) recommendationUnsorted(search map[string]float64) map[int]float64 {
	result := make(map[int]float64)
	for id := range docGroup.tfidf {
		//length
		length2 := 0.0
		for _, num := range docGroup.tfidf[id] {
			length2 = num * num
		}
		length2 = math.Sqrt(length2)
		length1 := 0.0
		cos := 0.0
		for token, tfidf := range search {
			if _, exists := docGroup.tfidf[id][token]; exists {
				cos += docGroup.tfidf[id][token] * tfidf //cosine p1
			}
			length1 += tfidf * tfidf
		}
		length1 = math.Sqrt(length1)
		if cos > 0.0 {
			cos = cos / (length1 * length2) //cosine p2
		}
		result[id] = cos
	}
	return result
}

func (docGroup *DocumentGroup) makeTFIDF() { //map[int]map[string]float64 {
	/*if len(docGroup.tfidf) > 0 {
		//return docGroup.tfidf
	}*/
	idf := docGroup.makeIdf()
	tf := docGroup.makeTf()
	tfidf := make(map[int]map[string]float64)
	for id, words := range tf {
		for word := range words {
			if _, exists := tfidf[id]; !exists {
				tfidf[id] = make(map[string]float64)
			}
			tfidf[id][word] += tf[id][word] * idf[word] //tf*idf
		}
	}
	docGroup.tfidf = tfidf
	//return docGroup.tfidf
}

func (docGroup *DocumentGroup) makeTf() map[int]map[string]float64 {
	tf := make(map[int]map[string]float64)
	for _, doc := range docGroup.DocPartials {
		tf[doc.Doc.Id] = doc.GetTF(len(docGroup.DocPartials))
	}
	return tf
}

func (docGroup *DocumentGroup) makeIdf() map[string]float64 {
	if len(docGroup.idf) > 0 {
		return docGroup.idf
	}
	idf := make(map[string]float64)
	for _, doc := range docGroup.DocPartials {
		for _, t := range doc.GetTokens(false) {
			if _, exists := idf[t]; !exists {
				idf[t] = 0
			}
			idf[t] += 1
		}
	}
	//length := float64(len(idf))
	numDocs := float64(len(docGroup.DocPartials))
	for id, count := range idf {
		//body := float64(len(docGroup.documents)) / count
		//idf[id] = 1 + math.Log(body) //idf
		//idf[id] = 1 + math.Log((count/100)/float64(len(docGroup.documents)))*math.Log(float64(len(docGroup.documents))/count) //idf
		idf[id] = numDocs / count
	}
	docGroup.idf = idf
	return idf
}

func fillFromDb() []*Document {

	/*
		TODO: Get from your database here

		...
			var docs []*Document

			for _, v := range result {
				docs = append(docs, v)
			}
		...
	*/

	dummyData := []*Document
	return dummy
}

func (doc *DocumentPartial) GetTokens(duplicates bool) []string {
	if len(doc.tokens) > 0 {
		if duplicates {
			return doc.tokensDuplicate
		}
		return doc.tokens
	}
	//Join title+keywords
	tmp := strings.Join(doc.Doc.Keywords, " ")
	tmp = doc.Doc.Title + " " + tmp

	tokens := strings.Split(doc.cleanTokenString(tmp), " ")

	doc.tokensDuplicate = tokens
	noDuplicate := make(map[string]bool)
	for _, word := range tokens {
		noDuplicate[word] = true
	}
	tokens2 := []string{}
	for word := range noDuplicate {
		tokens2 = append(tokens2, word)
	}
	doc.tokens = tokens2
	if duplicates {
		return doc.tokensDuplicate
	}
	return doc.tokens
}

func (doc *DocumentPartial) cleanTokenString(w string) string {

	/*
		TODO: Remove unwanted characters here, example below
	*/

	words := strings.Replace(w, ",", "", -1)
	words = strings.Replace(words, ".", "", -1)
	words = strings.Replace(words, ",", "", -1)
	words = strings.Replace(words, "&", "", -1)
	words = strings.Replace(words, "-", "", -1)
	words = strings.Replace(words, "!", "", -1)
	words = strings.Replace(words, ")", "", -1)
	words = strings.Replace(words, "(", "", -1)
	words = strings.Replace(words, "-", "", -1)

	words = strings.ToLower(words)

	return words
}
func (doc *DocumentPartial) GetTF(numDocs int) map[string]float64 {
	if len(doc.tf) > 0 {
		return doc.tf
	}
	tokens := doc.GetTokens(true)
	length := float64(len(tokens))
	tf := make(map[string]float64)
	for _, word := range tokens {
		if _, exists := tf[word]; !exists {
			tf[word] = 0
		}
		tf[word] = tf[word] + 1
	}
	for key := range tf {
		//body := (tf[key] / 100) / length
		//tf[key] = 1 + math.Log(body) //tf
		freq := tf[key]
		tf[key] = freq / length //f
	}
	doc.tf = tf
	return doc.tf
}
