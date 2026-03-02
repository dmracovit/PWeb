module.exports = function (eleventyConfig) {
  // Copy static assets to output
  eleventyConfig.addPassthroughCopy("src/assets");
  eleventyConfig.addPassthroughCopy("src/admin");

  // Sort collections by "order" field
  eleventyConfig.addCollection("menu", (collection) =>
    collection.getFilteredByTag("menu").sort((a, b) => a.data.order - b.data.order)
  );
  eleventyConfig.addCollection("testimonials", (collection) =>
    collection.getFilteredByTag("testimonials").sort((a, b) => a.data.order - b.data.order)
  );
  eleventyConfig.addCollection("gallery", (collection) =>
    collection.getFilteredByTag("gallery").sort((a, b) => a.data.order - b.data.order)
  );

  return {
    dir: {
      input: "src",
      output: "_site",
      includes: "_includes",
      data: "_data",
    },
  };
};
