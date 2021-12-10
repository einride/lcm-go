//go:build mage

package main

import (
	"github.com/magefile/mage/mg"

	// mage:import
	_ "github.com/einride/mage-tools/maketargets"
	// mage:import
	"github.com/einride/mage-tools/tools/common"
	// mage:import
	"github.com/einride/mage-tools/tools/gitverifynodiff"
	// mage:import
	"github.com/einride/mage-tools/tools/goreview"
	// mage:import
	"github.com/einride/mage-tools/tools/golangcilint"
	// mage:import
	"github.com/einride/mage-tools/tools/commitlint"
	// mage:import
	"github.com/einride/mage-tools/tools/prettier"
)

func All() {
	mg.Deps(
		mg.F(common.MockgenGenerate, "mockplayer", "test/mocks/player/service.go", "github.com/einride/lcm-go/pkg/player", "Transmitter"),
	)
	mg.Deps(
		mg.F(commitlint.Commitlint, "master"),
		golangcilint.GolangciLint,
		goreview.Goreview,
		common.GoTest,
	)
	mg.Deps(
		common.GoModTidy,
		prettier.FormatMarkdown,
	)
	mg.SerialDeps(
		gitverifynodiff.GitVerifyNoDiff,
	)
}
