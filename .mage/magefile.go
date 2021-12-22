//go:build mage

package main

import (
	"github.com/magefile/mage/mg"

	// mage:import
	_ "go.einride.tech/mage-tools/mgmake"
	// mage:import
	"go.einride.tech/mage-tools/targets/mgmockgen"
	// mage:import
	"go.einride.tech/mage-tools/targets/mggo"
	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
	// mage:import
	"go.einride.tech/mage-tools/targets/mggoreview"
	// mage:import
	"go.einride.tech/mage-tools/targets/mggolangcilint"
	// mage:import
	"go.einride.tech/mage-tools/targets/mgcommitlint"
	// mage:import
	"go.einride.tech/mage-tools/targets/mgprettier"
)

func All() {
	mg.Deps(
		mg.F(mgmockgen.MockgenGenerate, "mockplayer", "test/mocks/player/service.go", "github.com/einride/lcm-go/pkg/player", "Transmitter"),
	)
	mg.Deps(
		mg.F(mgcommitlint.Commitlint, "master"),
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
		mggo.GoTest,
	)
	mg.Deps(
		mggo.GoModTidy,
		mgprettier.FormatMarkdown,
	)
	mg.SerialDeps(
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
