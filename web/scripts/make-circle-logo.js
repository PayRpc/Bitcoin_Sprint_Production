const sharp = require('sharp');
const fs = require('fs');

const input = 'public/20250823_1017_Bitcoin Sprint Logo_.png';
const output = 'public/logo-bitcoin-sprint-circle.png';

(async () => {
  try {
    const image = sharp(input);
    const metadata = await image.metadata();
    const size = Math.min(metadata.width, metadata.height);
    const left = Math.floor((metadata.width - size) / 2);
    const top = Math.floor((metadata.height - size) / 2);

    // extract square center
    const sq = image.extract({ left, top, width: size, height: size }).resize(1024, 1024);

    // create circular mask
    const circle = Buffer.from(
      `<svg width="1024" height="1024"><circle cx="512" cy="512" r="512" fill="#fff"/></svg>`
    );

    await sq
      .composite([
        { input: circle, blend: 'dest-in' }
      ])
      .png({ quality: 100 })
      .toFile(output);

    console.log('Created', output);
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
})();
