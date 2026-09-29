package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/micro-editor/micro/v2/pkg/highlight"
)

// Load the real bundled rules and resolve their includes, just as the editor does.
func vueHighlighter(t *testing.T) *highlight.Highlighter {
	t.Helper()
	var files []*highlight.File
	var vue *highlight.Def
	for _, name := range []string{"vue", "javascript", "typescript", "css"} {
		data, err := os.ReadFile("../../runtime/syntax/" + name + ".yaml")
		if err != nil {
			t.Fatal(err)
		}
		file, err := highlight.ParseFile(data)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
		if name == "vue" {
			header, err := highlight.MakeHeaderYaml(data)
			if err != nil {
				t.Fatal(err)
			}
			vue, err = highlight.ParseDef(file, header)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	highlight.ResolveIncludes(vue, files)
	return highlight.NewHighlighter(vue)
}

func assertVueGroup(t *testing.T, input, token, want string) {
	t.Helper()
	index := strings.Index(input, token)
	if index < 0 {
		t.Fatalf("missing test token %q", token)
	}
	before := input[:index]
	line := strings.Count(before, "\n")
	column := len([]rune(before[strings.LastIndex(before, "\n")+1:]))
	matches := vueHighlighter(t).HighlightString(input)
	var got highlight.Group
	for i := 0; i <= column; i++ {
		if group, ok := matches[line][i]; ok {
			got = group
		}
	}
	expected, ok := highlight.Groups[want]
	if !ok {
		t.Fatalf("unknown highlight group %q", want)
	}
	if got != expected {
		t.Errorf("%q: got %s, want %s", token, got, want)
	}
}

func TestVueScript(t *testing.T) {
	for _, attrs := range []string{
		"", " setup", ` lang="js" setup`, ` setup data-label="a > b"`,
	} {
		t.Run("javascript"+attrs, func(t *testing.T) {
			input := "<script" + attrs + ">\nimport value from \"vue\";\nconst count = 42;\n</script>"
			assertVueGroup(t, input, "vue", "constant.string")
			assertVueGroup(t, input, "const", "statement")
			assertVueGroup(t, input, "42", "constant.number")
		})
	}
	for _, attrs := range []string{
		` lang="ts"`, ` setup lang="ts"`, ` lang="ts" setup`,
		` setup lang='ts'`, ` setup lang = "ts"`, "\tlang=\"ts\"\tsetup",
		` setup generic="T extends Map<string, number>" lang="ts"`,
		` lang="ts" generic="T extends Map<string, number>" setup`,
		` setup lang=ts`,
	} {
		t.Run("typescript"+attrs, func(t *testing.T) {
			input := "<script" + attrs + ">\nimport type { Component } from \"vue\";\nconst count: number = 42;\n</script>"
			assertVueGroup(t, input, "number = 42", "type")
			assertVueGroup(t, input, "vue", "constant.string")
			assertVueGroup(t, input, "42", "constant.number")
		})
	}
	// An attribute value containing lang=ts is not a TypeScript language selector.
	for _, attrs := range []string{` data-lang="ts"`, ` data-label=' lang="ts"'`} {
		t.Run("not-typescript"+attrs, func(t *testing.T) {
			input := "<script" + attrs + ">\nconst number = 42;\n</script>"
			assertVueGroup(t, input, "number", "default")
		})
	}
}

func TestVueStyle(t *testing.T) {
	for _, attrs := range []string{"", " scoped", " module", ` lang="css" scoped`, ` scoped lang="css"`} {
		t.Run(attrs, func(t *testing.T) {
			input := "<style" + attrs + ">\n.card { color: red; }\n</style>"
			assertVueGroup(t, input, "color", "type")
		})
	}
}

// The current engine skips parent patterns before nested strings. Keyword
// and attribute-name assertions below avoid depending on a fix to that engine.
// The fixture still includes ordinary imports and quoted attributes for manual testing.
func TestVueTemplate(t *testing.T) {
	data, err := os.ReadFile("testdata/vue.vue")
	if err != nil {
		t.Fatal(err)
	}
	input := string(data)
	for _, check := range []struct{ token, group string }{
		{"DropdownMenuItem", "symbol.tag"},
		{"disabled", "identifier"},
		{"\"visible\"", "constant.string"}, {"count > 0", "constant.string"},
		{"'select()'", "constant.string"}, {"\"menu\"", "constant.string"},
		{"my-component", "symbol.tag"},
		{"section", "symbol.tag"}, {"after-nested", "constant.string"},
		{"42", "constant.number"},
		{"Second line > still", "constant.string"},
		{"interface", "statement"},
		{"string;", "type"}, {"color:", "type"}, {"FakeComponent", "comment.block"},
		{"&amp;", "special"},
	} {
		t.Run(check.token, func(t *testing.T) {
			assertVueGroup(t, input, check.token, check.group)
		})
	}
}

func TestVueBlockOrder(t *testing.T) {
	blocks := []string{
		"<script setup lang=\"ts\">\nconst count: number = 42;\n</script>",
		"<template>\n<CustomButton title=\"hello\" />\n</template>",
		"<style scoped>\n.card { color: red; }\n</style>",
	}
	for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			input := blocks[order[0]] + "\n" + blocks[order[1]] + "\n" + blocks[order[2]]
			assertVueGroup(t, input, "number = 42", "type")
			assertVueGroup(t, input, "CustomButton", "symbol.tag")
			assertVueGroup(t, input, "hello", "constant.string")
			assertVueGroup(t, input, "color", "type")
		})
	}
}

func TestVueInlineBlocks(t *testing.T) {
	input := `<script setup lang="ts">const count: number = 42;</script><template><MyButton title="hello" /></template>`
	assertVueGroup(t, input, "const", "statement")
	assertVueGroup(t, input, "number", "type")
	assertVueGroup(t, input, "MyButton", "symbol.tag")
	assertVueGroup(t, input, "hello", "constant.string")
}

func TestVueAttributeNames(t *testing.T) {
	// Unquoted values exercise attribute rules without triggering the existing
	// engine bug that skips parent patterns before nested quoted strings.
	input := `<template><MyButton
  v-if=visible :data-inset=count @click.stop=select #default=slotProps
  class=menu data-test=value disabled
/></template>`
	for _, name := range []string{"v-if", ":data-inset", "@click.stop", "#default", "class", "data-test", "disabled"} {
		t.Run(name, func(t *testing.T) {
			assertVueGroup(t, input, name, "identifier")
		})
	}
}

func TestVueScriptLikeComponents(t *testing.T) {
	input := `<template>
  <script-panel lang="ts"><MyButton /></script-panel>
  <style-picker><OtherButton /></style-picker>
</template>`
	assertVueGroup(t, input, "MyButton", "symbol.tag")
	assertVueGroup(t, input, "OtherButton", "symbol.tag")
}
