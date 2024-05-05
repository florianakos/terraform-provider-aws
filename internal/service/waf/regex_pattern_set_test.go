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
func TestAccWAFRegexPatternSet_serial(t *testing.T) {
	t.Parallel()

	testCases := map[string]func(t *testing.T){
		"basic":          testAccRegexPatternSet_basic,
		"changePatterns": testAccRegexPatternSet_changePatterns,
		"noPatterns":     testAccRegexPatternSet_noPatterns,
		"disappears":     testAccRegexPatternSet_disappears,
	}

	acctest.RunSerialTests1Level(t, testCases, 0)
}

func testAccRegexPatternSet_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.RegexPatternSet
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_pattern_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexPatternSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexPatternSetConfig_basic(patternSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegexPatternSetExists(ctx, resourceName, &v),
					acctest.MatchResourceAttrGlobalARN(resourceName, "arn", "waf", regexache.MustCompile(`regexpatternset/.+`)),
					resource.TestCheckResourceAttr(resourceName, "name", patternSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_pattern_strings.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "one"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "two"),
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

func testAccRegexPatternSet_changePatterns(t *testing.T) {
	ctx := acctest.Context(t)
	var before, after awstypes.RegexPatternSet
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_pattern_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexPatternSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexPatternSetConfig_basic(patternSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexPatternSetExists(ctx, resourceName, &before),
					resource.TestCheckResourceAttr(resourceName, "name", patternSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_pattern_strings.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "one"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "two"),
				),
			},
			{
				Config: testAccRegexPatternSetConfig_changes(patternSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexPatternSetExists(ctx, resourceName, &after),
					resource.TestCheckResourceAttr(resourceName, "name", patternSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_pattern_strings.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "two"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "three"),
					resource.TestCheckTypeSetElemAttr(resourceName, "regex_pattern_strings.*", "four"),
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

func testAccRegexPatternSet_noPatterns(t *testing.T) {
	ctx := acctest.Context(t)
	var patternSet awstypes.RegexPatternSet
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_pattern_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexPatternSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexPatternSetConfig_nos(patternSetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRegexPatternSetExists(ctx, resourceName, &patternSet),
					resource.TestCheckResourceAttr(resourceName, "name", patternSetName),
					resource.TestCheckResourceAttr(resourceName, "regex_pattern_strings.#", "0"),
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

func testAccRegexPatternSet_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.RegexPatternSet
	patternSetName := fmt.Sprintf("tfacc-%s", sdkacctest.RandString(5))
	resourceName := "aws_waf_regex_pattern_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.WAFServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckRegexPatternSetDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccRegexPatternSetConfig_basic(patternSetName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRegexPatternSetExists(ctx, resourceName, &v),
					testAccCheckRegexPatternSetDisappears(ctx, &v),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckRegexPatternSetDisappears(ctx context.Context, set *awstypes.RegexPatternSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)

		wr := tfwaf.NewRetryer(conn)
		_, err := wr.RetryWithToken(ctx, func(token *string) (interface{}, error) {
			req := &waf.UpdateRegexPatternSetInput{
				ChangeToken:       token,
				RegexPatternSetId: set.RegexPatternSetId,
			}

			for _, pattern := range set.RegexPatternStrings {
				update := awstypes.RegexPatternSetUpdate{
					Action:             awstypes.ChangeActionDelete,
					RegexPatternString: aws.String(pattern),
				}
				req.Updates = append(req.Updates, update)
			}

			return conn.UpdateRegexPatternSet(ctx, req)
		})
		if err != nil {
			return fmt.Errorf("Failed updating WAF Regex Pattern Set: %s", err)
		}

		_, err = wr.RetryWithToken(ctx, func(token *string) (interface{}, error) {
			opts := &waf.DeleteRegexPatternSetInput{
				ChangeToken:       token,
				RegexPatternSetId: set.RegexPatternSetId,
			}
			return conn.DeleteRegexPatternSet(ctx, opts)
		})
		if err != nil {
			return fmt.Errorf("Failed deleting WAF Regex Pattern Set: %s", err)
		}

		return nil
	}
}

func testAccCheckRegexPatternSetExists(ctx context.Context, n string, v *awstypes.RegexPatternSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No WAF Regex Pattern Set ID is set")
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)
		resp, err := conn.GetRegexPatternSet(ctx, &waf.GetRegexPatternSetInput{
			RegexPatternSetId: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return err
		}

		if *resp.RegexPatternSet.RegexPatternSetId == rs.Primary.ID {
			*v = *resp.RegexPatternSet
			return nil
		}

		return fmt.Errorf("WAF Regex Pattern Set (%s) not found", rs.Primary.ID)
	}
}

func testAccCheckRegexPatternSetDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_waf_regex_pattern_set" {
				continue
			}

			conn := acctest.Provider.Meta().(*conns.AWSClient).WAFClient(ctx)
			resp, err := conn.GetRegexPatternSet(ctx, &waf.GetRegexPatternSetInput{
				RegexPatternSetId: aws.String(rs.Primary.ID),
			})

			if err == nil {
				if *resp.RegexPatternSet.RegexPatternSetId == rs.Primary.ID {
					return fmt.Errorf("WAF Regex Pattern Set %s still exists", rs.Primary.ID)
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

func testAccRegexPatternSetConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_pattern_set" "test" {
  name                  = "%s"
  regex_pattern_strings = ["one", "two"]
}
`, name)
}

func testAccRegexPatternSetConfig_changes(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_pattern_set" "test" {
  name                  = "%s"
  regex_pattern_strings = ["two", "three", "four"]
}
`, name)
}

func testAccRegexPatternSetConfig_nos(name string) string {
	return fmt.Sprintf(`
resource "aws_waf_regex_pattern_set" "test" {
  name = "%s"
}
`, name)
}
