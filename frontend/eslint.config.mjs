import { fixupConfigRules, fixupPluginRules } from "@eslint/compat";
import react from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import jsdoc from "eslint-plugin-jsdoc";
import globals from "globals";
import path from "node:path";
import { fileURLToPath } from "node:url";
import js from "@eslint/js";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all
});

export default [...fixupConfigRules(compat.extends(
    "eslint:recommended",
    "plugin:react/recommended",
    "plugin:react-hooks/recommended",
    "prettier",
)), {
    plugins: {
        react: fixupPluginRules(react),
        "react-hooks": fixupPluginRules(reactHooks),
        jsdoc,
    },

    languageOptions: {
        globals: {
            ...globals.browser,
            ...globals.node,
        },

        ecmaVersion: "latest",
        sourceType: "module",
    },

    settings: {
        react: {
            version: "detect",
        },
    },

    rules: {
        "react/prop-types": "off",
        "react/react-in-jsx-scope": "off",

        // JSDoc rules for documentation coverage
        "jsdoc/require-jsdoc": ["error", {
            require: {
                FunctionDeclaration: true,
                MethodDefinition: false,
                ClassDeclaration: true,
                ArrowFunctionExpression: false,
                FunctionExpression: false
            },
            contexts: [
                // Require JSDoc for exported arrow functions (React components)
                "ExportNamedDeclaration > VariableDeclaration > VariableDeclarator > ArrowFunctionExpression",
                "ExportDefaultDeclaration > ArrowFunctionExpression",
                // Require JSDoc for const arrow functions at module level (React components)
                "Program > VariableDeclaration > VariableDeclarator[id.name=/^[A-Z]/] > ArrowFunctionExpression"
            ]
        }],
        "jsdoc/require-description": "error",
        "jsdoc/require-param-description": "error",
        "jsdoc/require-returns-description": "error",
        "jsdoc/check-alignment": "error",
        "jsdoc/check-indentation": "off", // Can be strict, disable if too noisy
    },
}];
