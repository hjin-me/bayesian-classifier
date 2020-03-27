package bayesianc

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/hjin-me/bayesian-classifier/segmenter"
	"github.com/hjin-me/go-utils/logex"
)

type Score struct {
	items []*ScoreItem
}

type ScoreItem struct {
	Category string  `json:"category"` // 分类名称
	Score    float64 `json:"score"`    // 概率值
}
type data struct {
	Category map[string]float64            `json:"category"` // 分类数据
	Words    map[string]map[string]float64 `json:"words"`    // 单词数据
	Docs     map[string]bool               `json:"docs"`     // 文档数据
}
type SDK struct {
	debug     bool
	data      data
	segmenter segmenter.Segmenter
}

func (s *SDK) EnableDebug(b bool) {
	s.debug = b
}

func (s *SDK) LoadDictionary(r io.Reader) error {
	return s.segmenter.LoadDictionary(r)
}

func (s *SDK) LoadModel(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &s.data)
	if err != nil {
		return err
	}
	return nil
}
func (s SDK) calcSampleCounts() float64 {
	total := 0.0
	for _, n := range s.data.Category {
		total += n
	}
	return total
}
func (s SDK) calcSampleCountsByCategory(category string) float64 {
	return s.data.Category[category]
}
func (s SDK) calcWordCountsByCategory(word, category string) float64 {
	if set, ok := s.data.Words[word]; ok {
		return set[category]
	}
	return 0
}
func (s SDK) calcWordCounts(word string) float64 {
	var total float64 = 0
	for _, s := range s.data.Words[word] {
		total += s
	}
	return total
}
func (s SDK) factor(word, category string) float64 {
	totalCategoryCounts := s.calcSampleCounts()
	targetCategoryCounts := s.calcSampleCountsByCategory(category)
	wordCountsTotal := s.calcWordCounts(word)
	wordCountsInCategory := s.calcWordCountsByCategory(word, category)
	// 频数较小的作为废弃规则
	//if wordCountsTotal < 5 {
	//	//prob *= (1 / targetCategoryCounts) / (1 / totalCategoryCounts)
	//	continue
	//}

	//log.Printf("%s = %0.6f / %0.6f", word, wordCountsInCategory/targetCategoryCounts, wordCountsTotal/totalCategoryCounts)
	// 拉普拉斯平滑
	wordCountsInCategory += 1
	targetCategoryCounts += 2
	wordCountsTotal += 1
	totalCategoryCounts += 2
	//log.Printf("[%s], %s = %0.6f / %0.6f, laplace, = %0.6f", category, word, wordCountsInCategory/targetCategoryCounts, wordCountsTotal/totalCategoryCounts, (wordCountsInCategory/targetCategoryCounts)/(wordCountsTotal/totalCategoryCounts))
	if s.debug {
		logex.Debugf("factor %s[%s] = %0.6f = ( %0.6f / %0.6f ) / ( %0.6f / %0.6f )",
			word, category, wordCountsInCategory/targetCategoryCounts/(wordCountsTotal/totalCategoryCounts),
			wordCountsInCategory, targetCategoryCounts, wordCountsTotal, totalCategoryCounts)
	}
	return wordCountsInCategory / targetCategoryCounts / (wordCountsTotal / totalCategoryCounts)
}
func (s SDK) Categorize(b []byte) []ScoreItem {
	segments := s.segmenter.Segment(b)
	words := segmenter.SegmentsToSlice(segments, false)
	var result []ScoreItem
	for category, categoryCounts := range s.data.Category {
		prob := 1.0
		for _, word := range words {
			prob *= s.factor(word, category)
		}
		result = append(result, ScoreItem{
			Category: category,
			Score:    prob * categoryCounts / s.calcSampleCounts(),
		})
		if s.debug {
			logex.Debugf("P(%s) = %0.6f = %0.0f / %0.0f", category, categoryCounts/s.calcSampleCounts(), categoryCounts, s.calcSampleCounts())
		}
	}
	return result
}

func New() *SDK {
	s := SDK{}
	s.segmenter = segmenter.Segmenter{}
	return &SDK{}
}