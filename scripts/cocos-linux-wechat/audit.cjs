'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');

// Parse generated code rather than grep across minified expressions: a numeric
// Math.pow followed by an unrelated BigInt call is valid and must not fail.
function audit(root, { stripInlineMaps = false, previewCompatible = true } = {}) {
  const dependency = createRequire(path.join(process.env.COCOS_CLI_ROOT || __dirname, 'package.json'));
  const { parse } = dependency('@babel/parser');
  const traverse = dependency('@babel/traverse').default;
  const named = (node, name) => node?.type === 'Identifier' && node.name === name ||
    node?.type === 'StringLiteral' && node.value === name;
  let checked = 0;
  function walk(dir) {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const file = path.join(dir, item.name);
      if (item.isDirectory()) walk(file);
      else if (item.isFile() && file.endsWith('.js')) {
        const source = fs.readFileSync(file, 'utf8');
        const ast = parse(source, { sourceType: 'unambiguous' });
        function bigint(node, scope, visited = new Set()) {
          if (!node) return false;
          if (node.type === 'BigIntLiteral') return true;
          if (node.type === 'CallExpression' && node.callee.type === 'Identifier' && node.callee.name === 'BigInt') return true;
          if (node.type === 'Identifier' && !visited.has(node)) {
            visited.add(node);
            const binding = scope.getBinding(node.name);
            return binding?.path.isVariableDeclarator() ? bigint(binding.path.node.init, binding.path.scope, visited) : false;
          }
          return node.type === 'MemberExpression' && named(node.property, 'ORDER') && named(node.object?.property, 'Fn');
        }
        traverse(ast, {
          LogicalExpression(expr) {
            if (previewCompatible && expr.node.operator === '??')
              throw new Error('Generated preview-incompatible ?? in ' + path.relative(root, file));
          },
          OptionalMemberExpression(expr) {
            if (previewCompatible)
              throw new Error('Generated preview-incompatible optional chaining in ' + path.relative(root, file));
          },
          OptionalCallExpression(expr) {
            if (previewCompatible)
              throw new Error('Generated preview-incompatible optional call in ' + path.relative(root, file));
          },
          CallExpression(call) {
            const callee = call.node.callee;
            if (callee.type === 'MemberExpression' && callee.object.type === 'Identifier' &&
                callee.object.name === 'Math' && named(callee.property, 'pow') &&
                call.node.arguments.some(arg => bigint(arg, call.scope))) {
              throw new Error('Generated Math.pow receives BigInt in ' + path.relative(root, file));
            }
          }
        });
        // CLI4 can retain inline maps even with debug=false. Drop only an
        // actual trailing source-map comment, never a matching string literal.
        if (stripInlineMaps) {
          const comment = ast.comments?.find(comment =>
            /^#\s*sourceMappingURL=data:/.test(comment.value.trim()) &&
            source.slice(comment.end).trim() === '');
          if (comment) fs.writeFileSync(file, source.slice(0, comment.start) + '\n');
        }
        checked++;
      }
    }
  }
  walk(root);
  return checked;
}
module.exports = { audit };

// Re-run this exact AST audit on the WeChat IDE's *processed* JS directory,
// not merely the raw Cocos output. This cannot be replaced by a grep/CI green.
if (require.main === module) {
  const directory = process.argv[2];
  if (!directory || !fs.statSync(directory).isDirectory())
    throw new Error('Usage: node audit.cjs <actual-generated-or-WeChat-processed-directory>');
  console.log('Audited ' + audit(path.resolve(directory)) + ' JavaScript files; preview syntax and BigInt safe');
}
