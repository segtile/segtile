# annotation-tool

## Evaluation mode

For measuring annotation speed against existing ground truth, the tool can be started in evaluation mode:

```
annotation-tool --evaluation [--eval-threshold 0.95] <project-directory>
```

In this mode:

- Final masks in the project output directory are treated as ground truth (GT); every annotation starts from an empty mask and nothing is ever saved.
- A second window shows the original image overlaid with the GT mask in red. Once the dice score between the current annotation and the GT reaches the threshold (default 0.95), the overlay turns green.
- Every second image (odd indices) is locked to the manual-fix brush, so half of the annotations are made with and half without superpixel support, interleaved to cancel out learning effects.
- The time per image is kept in memory. When the user navigates away from an image, its duration, mode, and dice score are printed to the console immediately, so a log survives even an abnormal exit. On exit, a summary is printed with per-mode mean time and mean dice score — once for images that met the threshold and once for all images.

