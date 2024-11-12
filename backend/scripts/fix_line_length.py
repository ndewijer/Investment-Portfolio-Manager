"""Script to automatically fix line length issues."""

import os

import autopep8


def fix_python_files(directory: str) -> None:
    """Walk through directory and fix Python files."""
    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith('.py'):
                filepath = os.path.join(root, file)
                with open(filepath, 'r') as f:
                    content = f.read()

                fixed = autopep8.fix_code(
                    content,
                    options={'max_line_length': 100}
                )

                with open(filepath, 'w') as f:
                    f.write(fixed)


if __name__ == '__main__':
    fix_python_files('app')
