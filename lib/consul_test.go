package lib

import "testing"

func TestBuildTagShouldReturnWellFormatedString(t *testing.T) {
	tag := Tag{Key: "toto", Value: "titi"}
	result := tag.BuildTag()
	if result != "toto=titi" {
		t.Error("Build Tag shoudl return toto=titi and it returned", result)
	}
}

func TestDesconstructTagShouldBuildAGoodTagFromString(t *testing.T) {
	tagStr := "toto=titi"
	//var tag *Tag = &Tag{}
	tag := &Tag{}
	tag.DeconstructTag(tagStr)
	if tag.Key != "toto" {
		t.Error("Key should be toto and is : ", tag.Key)
	}
	if tag.Value != "titi" {
		t.Error("Value should be titi and is : ", tag.Value)
	}
}
