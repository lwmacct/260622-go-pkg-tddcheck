package filelayout

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rulekit"
)

func unclassifiedFileViolation(profile rulekit.Profile, file rulekit.GoFile) []Violation {
	if !profile.WarnUnclassifiedFiles || file.IsTest || file.Layer != "" {
		return nil
	}
	return []Violation{{
		File:     rulekit.DisplayFilename(file.AbsPath),
		Line:     1,
		Code:     RuleID + "/unclassified-file",
		Severity: rulekit.SeverityWarning,
		Message:  "file is outside configured architecture layers",
	}}
}

func subjectPrivateAccessViolations(snapshot *rulekit.Snapshot) []Violation {
	if !snapshot.Profile.WarnSubjectPrivateAccess {
		return nil
	}
	filesByPath := make(map[string]rulekit.GoFile, len(snapshot.Files))
	for _, file := range snapshot.Files {
		filesByPath[file.AbsPath] = file
	}

	type accessKey struct {
		file   string
		object string
	}
	seen := map[accessKey]bool{}
	var violations []Violation
	for _, file := range snapshot.Files {
		if file.IsTest || file.Layer == "" || file.TypesInfo == nil || !businessIdentity(file) {
			continue
		}
		for identifier, object := range file.TypesInfo.Uses {
			if identifier == nil || object == nil || object.Exported() || object.Pkg() == nil || object.Pkg().Path() != file.PackagePath {
				continue
			}
			position := file.Fset.PositionFor(object.Pos(), true)
			owner, ok := filesByPath[position.Filename]
			if !ok || owner.AbsPath == file.AbsPath || owner.Layer != file.Layer || !businessIdentity(owner) || owner.Identity.Subject == file.Identity.Subject {
				continue
			}
			key := accessKey{file: file.AbsPath, object: object.Pkg().Path() + "\x00" + object.Name() + "\x00" + position.Filename}
			if seen[key] {
				continue
			}
			seen[key] = true
			violations = append(violations, Violation{
				File:     rulekit.DisplayFilename(file.AbsPath),
				Line:     file.Fset.PositionFor(identifier.Pos(), true).Line,
				Code:     RuleID + "/subject-private-access",
				Severity: rulekit.SeverityWarning,
				Message: fmt.Sprintf(
					"%s subject %q uses private declaration %s owned by subject %q; move shared implementation to x.shared.*",
					file.Layer,
					file.Identity.Subject,
					object.Name(),
					owner.Identity.Subject,
				),
			})
		}
	}
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		return violations[i].Message < violations[j].Message
	})
	return violations
}

func businessIdentity(file rulekit.GoFile) bool {
	return file.IdentityOK &&
		file.Identity.Subject != "" &&
		file.Identity.Namespace == "" &&
		file.Identity.Kind != "free"
}

func sharedDeclarationLineViolations(file rulekit.GoFile, name fileName, maximum int) []Violation {
	if maximum <= 0 || name.Namespace != "shared" {
		return nil
	}
	lines, first := allDeclarationLines(file.Fset, file.AST)
	if lines <= maximum {
		return nil
	}
	violation := violationAt(
		file.Fset,
		file.AbsPath,
		first,
		fmt.Sprintf("x.shared declarations occupy %d lines (maximum %d); prefer subject-owned files when declarations are not genuinely shared", lines, maximum),
	)
	violation.Code = RuleID + "/shared-size"
	violation.Severity = rulekit.SeverityWarning
	return []Violation{violation}
}

func allDeclarationLines(fileSet *token.FileSet, parsedFile *ast.File) (int, token.Pos) {
	var lines int
	var first token.Pos
	for _, declaration := range parsedFile.Decls {
		genDecl, isGen := declaration.(*ast.GenDecl)
		if !isGen || (genDecl.Tok != token.TYPE && genDecl.Tok != token.CONST && genDecl.Tok != token.VAR) {
			continue
		}
		if first == token.NoPos {
			first = declaration.Pos()
		}
		start := fileSet.PositionFor(declaration.Pos(), true).Line
		end := fileSet.PositionFor(declaration.End(), true).Line
		if end >= start {
			lines += end - start + 1
		}
	}
	return lines, first
}
