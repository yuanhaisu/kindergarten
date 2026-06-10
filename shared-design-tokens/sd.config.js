const StyleDictionary = require('style-dictionary');
const tokens = require('./tokens.json');

// ============================================================
// Deep merge helper — merge shared tokens with platform-specific ones
// ============================================================
function deepMerge(target, source) {
  const result = { ...target };
  for (const key in source) {
    if (
      source[key] &&
      typeof source[key] === 'object' &&
      !Array.isArray(source[key]) &&
      !source[key].hasOwnProperty('value')
    ) {
      result[key] = deepMerge(result[key] || {}, source[key]);
    } else {
      result[key] = source[key];
    }
  }
  return result;
}

// ============================================================
// Custom Transforms
// ============================================================

// rpx unit conversion (teacher/parent mini-program)
StyleDictionary.registerTransform({
  name: 'size/rpx',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.unit && typeof token.value === 'number';
  },
  transformer: (token) => {
    return `${token.value * 2}rpx`;
  },
});

// px unit conversion (admin web)
StyleDictionary.registerTransform({
  name: 'size/px',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.unit && typeof token.value === 'number';
  },
  transformer: (token) => {
    return `${token.value}px`;
  },
});

// fontFamily array → CSS string (for SCSS output)
StyleDictionary.registerTransform({
  name: 'fontFamily/css',
  type: 'value',
  matcher: (token) => {
    return token.attributes && token.attributes.category === 'typography' &&
           token.attributes.type === 'fontFamily';
  },
  transformer: (token) => {
    if (Array.isArray(token.value)) {
      return token.value.map(f => `'${f}'`).join(', ');
    }
    return token.value;
  },
});

// ============================================================
// Custom Format — Tailwind ESM export
// ============================================================
StyleDictionary.registerFormat({
  name: 'tailwind/esm',
  formatter: function ({ dictionary }) {
    const p = dictionary.properties;

    const fontFamilyBase = p.typography && p.typography.fontFamily && p.typography.fontFamily.base
      ? p.typography.fontFamily.base.value
      : [];

    const output = {
      colors: {},
      borderRadius: {},
      boxShadow: {},
      fontFamily: {
        sans: Array.isArray(fontFamilyBase) ? fontFamilyBase : [fontFamilyBase],
      },
      fontSize: {},
      fontWeight: {},
      spacing: {},
    };

    // Flatten color hierarchy for Tailwind
    if (p.color) {
      for (const [category, typeMap] of Object.entries(p.color)) {
        for (const [name, token] of Object.entries(typeMap)) {
          if (!output.colors[category]) {
            output.colors[category] = {};
          }
          output.colors[category][name] = token.value;
        }
      }
    }

    // borderRadius
    if (p.borderRadius) {
      for (const [name, token] of Object.entries(p.borderRadius)) {
        output.borderRadius[name] = token.value;
      }
    }

    // boxShadow
    if (p.shadow) {
      for (const [name, token] of Object.entries(p.shadow)) {
        output.boxShadow[name] = token.value;
      }
    }

    // fontSize
    if (p.typography && p.typography.fontSize) {
      for (const [name, token] of Object.entries(p.typography.fontSize)) {
        output.fontSize[name] = token.value;
      }
    }

    // fontWeight
    if (p.typography && p.typography.fontWeight) {
      for (const [name, token] of Object.entries(p.typography.fontWeight)) {
        output.fontWeight[name] = token.value;
      }
    }

    // spacing
    if (p.spacing) {
      for (const [name, token] of Object.entries(p.spacing)) {
        output.spacing[name] = token.value;
      }
    }

    return `// Auto-generated from tokens.json — DO NOT EDIT MANUALLY\nexport default ${JSON.stringify(output, null, 2)} as const;\n`;
  },
});

// ============================================================
// Platform-specific builds
// ============================================================

const platforms = {
  teacher: {
    transforms: ['attribute/cti', 'name/cti/kebab', 'size/rpx', 'fontFamily/css'],
    format: 'scss/variables',
    dest: 'scss/tokens-teacher.scss',
  },
  parent: {
    transforms: ['attribute/cti', 'name/cti/kebab', 'size/rpx', 'fontFamily/css'],
    format: 'scss/variables',
    dest: 'scss/tokens-parent.scss',
  },
  admin: {
    transforms: ['attribute/cti', 'name/cti/camel', 'size/px'],
    format: 'tailwind/esm',
    dest: 'ts/tokens-admin.ts',
  },
};

Object.entries(platforms).forEach(([platform, config]) => {
  const mergedTokens = deepMerge(tokens.shared, tokens[platform] || {});

  const sd = StyleDictionary.extend({
    tokens: mergedTokens,
    platforms: {
      [platform]: {
        transforms: config.transforms,
        buildPath: 'build/',
        files: [
          {
            destination: config.dest,
            format: config.format,
          },
        ],
      },
    },
  });

  sd.buildAllPlatforms();
});
