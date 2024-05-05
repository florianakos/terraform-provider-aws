// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package waf_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/waf"
	awstypes "github.com/aws/aws-sdk-go-v2/service/waf/types"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	tfwaf "github.com/hashicorp/terraform-provider-aws/internal/service/waf"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// Serialized acceptance tests due to WAF account limits
// https://docs.aws.amazon.com/waf/latest/developerguide/limits.html
func TestAccWAFRegexMatchSet_serial(t *testing.T) {
	t.Parallel()

	testCases := map[string]func(t *testing.T){
		"basic":          testAccRegexMatchSet_basic,
		"changePatterns": testAccRegexMatchSet_changePatterns,
		"noPatterns":     testAccRegexMatchSet_noPatterns,
		"disappears":     testAccRegexMatchSet_disappears,
	}

	acctest.RunSerialTests1Level(t, testCases, 0)
}

func testAccRegexMatchSet_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var matchSet awstypes.RegexMatchSet
	var patternSet awstypes.RegexPatternSet
	var idx int

	matchSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_match_set.test"

	fieldToMatch := awstypes.FieldToMatch{
		Data: aws.String("User-Agent"),
		Type: awstypes.MatchFieldType("HEADER"),
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexMatchSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexMatchSetConfig_basic(matchSetName, patternSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegexMatchSetExists(ctx, resourceName, &matchSet),
					testAccCheckRegexPatternSetExists(ctx, "aws_waf_regex_pattern_set.test", &patternSet),
					acctest.MatchResourceAttrGlobalARN(resourceName, "arn", "waf", regexache.MustCompile(`regexmatchset/.+`)),
					computeRegexMatchSetTuple(&patternSet, &fieldToMatch, "NONE", &idx),
					resource.TestCheckResourceAttr(resourceName, "name", matchSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_match_tuple.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "regex_match_tuple.*", map[string]string{
						"field_to_match.#":      "1",
						"field_to_match.0.data": "user-agent",
						"field_to_match.0.type": "HEADER",
						"text_transformation":   "NONE",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRegexMatchSet_changePatterns(t *testing.T) {
	ctx := acctest.Context(t)
	var before, after awstypes.RegexMatchSet
	var patternSet awstypes.RegexPatternSet
	var idx1, idx2 int

	matchSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_match_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexMatchSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexMatchSetConfig_basic(matchSetName, patternSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexMatchSetExists(ctx, resourceName, &before),
					testAccCheckRegexPatternSetExists(ctx, "aws_waf_regex_pattern_set.test", &patternSet),
					computeRegexMatchSetTuple(&patternSet, &awstypes.FieldToMatch{Data: aws.String("User-Agent"), Type: awstypes.MatchFieldType("HEADER")}, "NONE", &idx1),
					resource.TestCheckResourceAttr(resourceName, "name", matchSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_match_tuple.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "regex_match_tuple.*", map[string]string{
						"field_to_match.#":      "1",
						"field_to_match.0.data": "user-agent",
						"field_to_match.0.type": "HEADER",
						"text_transformation":   "NONE",
					}),
				),
			},
			{
				Config: testAccRegexMatchSetConfig_changePatterns(matchSetName, patternSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexMatchSetExists(ctx, resourceName, &after),
					resource.TestCheckResourceAttr(resourceName, "name", matchSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_match_tuple.#", "1"),

					computeRegexMatchSetTuple(&patternSet, &awstypes.FieldToMatch{Data: aws.String("Referer"), Type: awstypes.MatchFieldType("HEADER")}, "COMPRESS_WHITE_SPACE", &idx2),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "regex_match_tuple.*", map[string]string{
						"field_to_match.#":      "1",
						"field_to_match.0.data": "referer",
						"field_to_match.0.type": "HEADER",
						"text_transformation":   "COMPRESS_WHITE_SPACE",
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRegexMatchSet_noPatterns(t *testing.T) {
	ctx := acctest.Context(t)
	var matchSet awstypes.RegexMatchSet
	matchSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_match_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexMatchSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexMatchSetConfig_noPatterns(matchSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexMatchSetExists(ctx, resourceName, &matchSet),
					resource.TestCheckResourceAttr(resourceName, "name", matchSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_match_tuple.#", "0"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccRegexMatchSet_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var matchSet awstypes.RegexMatchSet
	matchSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_match_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexMatchSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexMatchSetConfig_basic(matchSetName, patternSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegexMatchSetExists(ctx, resourceName, &matchSet),
					testAccCheckRegexMatchSetDisappears(ctx, &matchSet),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func computeRegexMatchSetTuple(patternSet *awstypes.RegexPatternSet, fieldToMatch *awstypes.FieldToMatch, textTransformation string, idx *int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := map[string]interface{}{
			"field_to_match":       tfwaf.FlattenFieldToMatch(fieldToMatch),
			"regex_pattern_set_id": *patternSet.RegexPatternSetId,
			"text_transformation":  textTransformation,
		}

		*idx = tfwaf.RegexMatchSetTupleHash(m)

		return nil
	}
}

func testAccCheckRegexMatchSetDisappears(ctx context.Context, set *awstypes.RegexMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)

		wr := tfwaf.NewRetryer(conn)
		_, err := wr.RetryWithToken(ctx, func(token *string) (interface{}, error) {
			req := &waf.UpdateRegexMatchSetInput{
				ChangeToken:     token,
				RegexMatchSetId: set.RegexMatchSetId,
			}

			for _, tuple := range set.RegexMatchTuples {
				req.Updates = append(req.Updates, awstypes.RegexMatchSetUpdate{
					Action:          awstypes.ChangeAction("DELETE"),
					RegexMatchTuple: &tuple,
				})
			}

			return conn.UpdateRegexMatchSet(ctx, req)
		})
		if err != nil {
			return fmt.Errorf("Failed updating WAF Regex Match Set: %s", err)
		}

		_, err = wr.RetryWithToken(ctx, func(token *string) (interface{}, error) {
			opts := &waf.DeleteRegexMatchSetInput{
				ChangeToken:     token,
				RegexMatchSetId: set.RegexMatchSetId,
			}
			return conn.DeleteRegexMatchSet(ctx, opts)
		})
		if err != nil {
			return fmt.Errorf("Failed deleting WAF Regex Match Set: %s", err)
		}

		return nil
	}
}

func testAccCheckRegexMatchSetExists(ctx context.Context, n string, v *awstypes.RegexMatchSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF Regex Match Set ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)
		resp, err := conn.GetRegexMatchSet(ctx, &waf.GetRegexMatchSetInput{
			RegexMatchSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.RegexMatchSet.RegexMatchSetId == rs.Primary.ID {
			*v = *resp.RegexMatchSet
			return nil
		}

		return fmt.Errorf("WAF Regex Match Set (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckRegexMatchSetDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_waf_regex_match_set" {
				continue
			}

			conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)
			resp, err := conn.GetRegexMatchSet(ctx, &waf.GetRegexMatchSetInput{
				RegexMatchSetId: aws.String(rs.Primary.ID),
			})

			if err == nil {
				if *resp.RegexMatchSet.RegexMatchSetId == rs.Primary.ID {
					return fmt.Errorf("WAF Regex Match Set %s still exists", rs.Primary.ID)
				}
			}

			// Return nil if the Regex Pattern Set is already destroyed
			if errs.IsA[*awstypes.WAFNonexistentItemException](err) {
				return nil
			}

			return err
		}

		return nil
	}
}

func testAccRegexMatchSetConfig_basic(matchSetName, patternSetName string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_match_set" "test" {
  name = "%s"

  regex_match_tuple {
    field_to_match {
      data = "User-Agent"
      type = "HEADER"
    }

    regex_pattern_set_id = aws_waf_regex_pattern_set.test.id
    text_transformation  = "NONE"
  }
}

resource "aws_waf_regex_pattern_set" "test" {
  name                  = "%s"
  regex_pattern_strings = ["one", "two"]
}
`, matchSetName, patternSetName)
}

func testAccRegexMatchSetConfig_changePatterns(matchSetName, patternSetName string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_match_set" "test" {
  name = "%s"

  regex_match_tuple {
    field_to_match {
      data = "Referer"
      type = "HEADER"
    }

    regex_pattern_set_id = aws_waf_regex_pattern_set.test.id
    text_transformation  = "COMPRESS_WHITE_SPACE"
  }
}

resource "aws_waf_regex_pattern_set" "test" {
  name                  = "%s"
  regex_pattern_strings = ["one", "two"]
}
`, matchSetName, patternSetName)
}

func testAccRegexMatchSetConfig_noPatterns(matchSetName string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_match_set" "test" {
  name = "%s"
}
`, matchSetName)
}
