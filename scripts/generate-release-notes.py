#!/usr/bin/env python3
"""
Generate GitHub release notes from conventional commits.

Parses git log between tags, categorizes commits by type (feat, fix, etc.),
extracts PR numbers, and formats as markdown suitable for GitHub releases.
"""

import subprocess
import sys
import re
import argparse
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass

# Category order for output (type -> display name)
CATEGORY_ORDER = [
    ("feat", "Features"),
    ("fix", "Bug Fixes"),
    ("refactor", "Refactoring"),
    ("perf", "Performance Improvements"),
    ("test", "Tests"),
    ("docs", "Documentation"),
    ("ci", "CI/CD"),
    ("chore", "Chores"),
]

# Regex pattern for conventional commits: type(scope): description
CONVENTIONAL_COMMIT_REGEX = r'^([a-z]+)(?:\(([^)]+)\))?: (.+)$'


@dataclass
class Commit:
    """Structured representation of a git commit."""
    hash: str
    subject: str
    body: str
    type: Optional[str] = None
    scope: Optional[str] = None
    description: str = ""
    pr_number: Optional[int] = None


class ReleaseNotesGenerator:
    """Orchestrates release note generation from git commits."""

    def __init__(self, version: str, repo_url: str, docs_url: str):
        """
        Initialize the release notes generator.

        Args:
            version: The release version tag (e.g., v0.2.2-beta)
            repo_url: GitHub repository URL
            docs_url: Documentation URL to include in footer
        """
        self.version = version
        self.repo_url = repo_url.rstrip('/')
        self.docs_url = docs_url

    def _find_previous_tag(self) -> Optional[str]:
        """
        Find the previous release tag before current version.

        Returns:
            Previous tag name, or None if this is the first release
        """
        try:
            result = subprocess.run(
                ['git', 'describe', '--tags', '--abbrev=0', f'{self.version}^'],
                capture_output=True,
                text=True,
                check=True
            )
            return result.stdout.strip()
        except subprocess.CalledProcessError:
            # No previous tag found (first release)
            return None

    def _get_commits(self, previous_tag: Optional[str]) -> List[str]:
        """
        Get commit log between previous tag and current version.

        Args:
            previous_tag: Previous release tag, or None for first release

        Returns:
            List of commit strings in format: hash|subject|body
        """
        # Build git log command
        if previous_tag:
            commit_range = f'{previous_tag}..{self.version}'
        else:
            # First release - get all commits up to version
            commit_range = self.version

        try:
            result = subprocess.run(
                [
                    'git', 'log',
                    '--format=%H|%s|%b',
                    '--no-merges',
                    commit_range
                ],
                capture_output=True,
                text=True,
                check=True
            )

            # Split by double newline (separates commits)
            commits = [c.strip() for c in result.stdout.split('\n\n') if c.strip()]
            return commits

        except subprocess.CalledProcessError as e:
            print(f"Error getting commits: {e.stderr}", file=sys.stderr)
            return []

    def _parse_commit(self, commit_text: str) -> Optional[Commit]:
        """
        Parse raw commit text into structured Commit object.

        Args:
            commit_text: Raw commit in format hash|subject|body

        Returns:
            Commit object, or None if parsing fails
        """
        # Split by delimiter: hash|subject|body
        parts = commit_text.strip().split('|', 2)
        if len(parts) < 2:
            return None

        hash_val = parts[0]
        subject = parts[1]
        body = parts[2] if len(parts) > 2 else ""

        commit = Commit(hash=hash_val, subject=subject, body=body)

        # Try to parse as conventional commit
        match = re.match(CONVENTIONAL_COMMIT_REGEX, subject)
        if match:
            commit.type = match.group(1)
            commit.scope = match.group(2) if match.group(2) else None
            commit.description = match.group(3)
        else:
            # Not a conventional commit - use full subject as description
            commit.description = subject

        # Extract PR number from subject (format: #123)
        pr_match = re.search(r'#(\d+)', subject)
        if pr_match:
            commit.pr_number = int(pr_match.group(1))

        return commit

    def _extract_body_excerpt(self, body: str) -> str:
        """
        Extract first meaningful paragraph from commit body.

        Skips metadata lines like Co-Authored-By and Signed-off-by.

        Args:
            body: Raw commit body text

        Returns:
            Cleaned excerpt (max 250 chars), or empty string
        """
        if not body:
            return ""

        # Split into paragraphs
        paragraphs = re.split(r'\n\n+', body.strip())

        for para in paragraphs:
            # Skip metadata lines
            if re.match(r'^(Co-Authored-By|Signed-off-by|Assisted-by):', para, re.IGNORECASE):
                continue

            # Skip bullet point lists (can be added in future enhancement)
            if re.match(r'^\s*[*-] ', para):
                continue

            # Clean up whitespace
            cleaned = ' '.join(para.split())

            # Return if meaningful content (at least 20 chars)
            if len(cleaned) > 20:
                # Truncate to 250 chars
                return cleaned[:250]

        return ""

    def _categorize_commits(self, commits: List[Commit]) -> Dict[str, List[Commit]]:
        """
        Group commits by type (feat, fix, etc.).

        Args:
            commits: List of parsed Commit objects

        Returns:
            Dictionary mapping commit type to list of commits
        """
        categorized: Dict[str, List[Commit]] = {}

        for commit in commits:
            # Use commit type if available, otherwise put in "other"
            category = commit.type if commit.type else "other"

            if category not in categorized:
                categorized[category] = []

            categorized[category].append(commit)

        return categorized

    def _format_commit(self, commit: Commit) -> str:
        """
        Format a commit as a markdown list item.

        Args:
            commit: Commit object to format

        Returns:
            Markdown-formatted string
        """
        # Build prefix: **type(scope)**: or **type**:
        if commit.type:
            if commit.scope:
                prefix = f"**{commit.type}({commit.scope})**:"
            else:
                prefix = f"**{commit.type}**:"
        else:
            prefix = ""

        # Build PR link if available
        pr_link = ""
        if commit.pr_number:
            pr_url = f"{self.repo_url}/pull/{commit.pr_number}"
            pr_link = f" [#{commit.pr_number}]({pr_url})"

        # Main line
        if prefix:
            result = f"- {prefix} {commit.description}{pr_link}"
        else:
            result = f"- {commit.description}{pr_link}"

        # Add body excerpt as blockquote if available
        excerpt = self._extract_body_excerpt(commit.body)
        if excerpt:
            result += f"\n  > {excerpt}"

        return result

    def _generate_changelog_section(self, categorized: Dict[str, List[Commit]]) -> str:
        """
        Generate the "What's Changed" markdown section.

        Args:
            categorized: Dictionary of commits grouped by type

        Returns:
            Markdown-formatted changelog
        """
        if not categorized:
            return "## What's Changed\n\nNo changes in this release.\n"

        sections = ["## What's Changed\n"]

        # Add sections in defined order
        for type_key, display_name in CATEGORY_ORDER:
            if type_key in categorized:
                sections.append(f"### {display_name}\n")

                for commit in categorized[type_key]:
                    sections.append(self._format_commit(commit))

                sections.append("")  # Blank line after each category

        # Add "Other Changes" section for non-conventional commits
        if "other" in categorized:
            sections.append("### Other Changes\n")
            for commit in categorized["other"]:
                sections.append(self._format_commit(commit))
            sections.append("")

        return "\n".join(sections)

    def _generate_footer(self) -> str:
        """
        Generate footer with documentation link.

        Returns:
            Markdown-formatted footer
        """
        return f"---\n\n**Documentation**: {self.docs_url}\n"

    def generate(self) -> str:
        """
        Main entry point: generate complete release notes.

        Returns:
            Complete markdown release notes
        """
        # 1. Find previous tag
        previous_tag = self._find_previous_tag()

        if previous_tag:
            print(f"INFO: Generating changelog from {previous_tag} to {self.version}", file=sys.stderr)
        else:
            print(f"INFO: No previous tag found. Using all commits up to {self.version}", file=sys.stderr)

        # 2. Get commits in range
        commit_texts = self._get_commits(previous_tag)

        if not commit_texts:
            print("INFO: No commits found in range.", file=sys.stderr)
            return f"## What's Changed\n\nNo changes in this release.\n\n{self._generate_footer()}"

        print(f"INFO: Found {len(commit_texts)} commits", file=sys.stderr)

        # 3. Parse commits
        commits = []
        for commit_text in commit_texts:
            commit = self._parse_commit(commit_text)
            if commit:
                commits.append(commit)
            else:
                print(f"WARNING: Could not parse commit: {commit_text[:50]}...", file=sys.stderr)

        # 4. Categorize commits
        categorized = self._categorize_commits(commits)

        # 5. Generate changelog section
        changelog = self._generate_changelog_section(categorized)

        # 6. Append footer
        footer = self._generate_footer()

        # 7. Return complete release notes
        return f"{changelog}\n{footer}"


def main():
    """CLI entry point."""
    parser = argparse.ArgumentParser(
        description="Generate GitHub release notes from conventional commits"
    )
    parser.add_argument(
        "--version",
        required=True,
        help="Release version tag (e.g., v0.2.2-beta)"
    )
    parser.add_argument(
        "--repo-url",
        required=True,
        help="GitHub repository URL (e.g., https://github.com/org/repo)"
    )
    parser.add_argument(
        "--docs-url",
        required=True,
        help="Documentation URL to include in footer"
    )

    args = parser.parse_args()

    try:
        generator = ReleaseNotesGenerator(args.version, args.repo_url, args.docs_url)
        release_notes = generator.generate()
        print(release_notes)
        sys.exit(0)
    except Exception as e:
        print(f"ERROR: Failed to generate release notes: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
