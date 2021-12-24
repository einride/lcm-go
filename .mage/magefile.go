//go:build mage
// +build mage

package main

import (
	"github.com/magefile/mage/mg"

	// mage:import
	_ "go.einride.tech/mage-tools/mgmake"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgcommitlint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggo"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggolangcilint"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggoreview"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgmockgen"

	// mage:import
	"go.einride.tech/mage-tools/targets/mgprettier"

	// mage:import
	"go.einride.tech/mage-tools/targets/mggitverifynodiff"
)

func All() {
	mg.Deps(
		mg.F(
			mgmockgen.MockgenGenerate,
			"mockplayer",
			"test/mocks/player/service.go",
			"go.einride.tech/lcm/pkg/player",
			"Transmitter",
		),
		mg.F(mgcommitlint.Commitlint, "master"),
		mgprettier.FormatMarkdown,
	)
	mg.Deps(
		mggolangcilint.GolangciLint,
		mggoreview.Goreview,
		mggo.GoTest,
	)
	mg.SerialDeps(
		mggo.GoModTidy,
		mggitverifynodiff.GitVerifyNoDiff,
	)
}
