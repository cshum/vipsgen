# VIPS Optional Outputs Examples

This example demonstrates how to capture optional output parameters from various VIPS operations. These optional outputs provide valuable information about the operations performed, such as transformation parameters, attention coordinates, and position data.

## Examples

### Smartcrop with Attention Coordinates

**Operation**: `vips_smartcrop`  
**Optional Outputs**: `AttentionX`, `AttentionY`

The coordinates where the algorithm detected the most interesting content for cropping.

```go
options := vips.DefaultSmartcropOptions()
err := img.Smartcrop(width, height, options)

// Access the attention coordinates
fmt.Printf("Attention Center: x=%d, y=%d\n", options.AttentionX, options.AttentionY)
```

### Mosaic with Transformation Parameters

**Operation**: `vips_mosaic`  
**Optional Outputs**: `Dx0`, `Dy0`, `Scale1`, `Angle1`, `Dx1`, `Dy1`

The detected transformation parameters when combining two images.

```go
options := vips.DefaultMosaicOptions()
err := img1.Mosaic(img2, direction, xref, yref, xsec, ysec, options)

// Access transformation parameters
fmt.Printf("Integer Offset: dx0=%d, dy0=%d\n", options.Dx0, options.Dy0)
fmt.Printf("Detected Scale: %.3f\n", options.Scale1)
fmt.Printf("Detected Rotation: %.3f degrees\n", options.Angle1)
```

### Max/Min with Position Coordinates

**Operations**: `vips_max`, `vips_min`  
**Optional Outputs**: `X`, `Y`

The coordinates where the maximum or minimum pixel values are located.

```go
maxOptions := vips.DefaultMaxOptions()
maxValue, err := img.Max(maxOptions)

// Access position of maximum value
fmt.Printf("Maximum value: %.2f at position x=%d, y=%d\n", 
    maxValue, maxOptions.X, maxOptions.Y)
```

### Draw Flood with Affected Area

**Operation**: `vips_draw_flood`  
**Optional Outputs**: `Left`, `Top`, `Width`, `Height`

The bounding box of the area affected by the flood fill operation.

```go
options := vips.DefaultDrawFloodOptions()
err := img.DrawFlood(color, x, y, options)

// Access affected area bounds
fmt.Printf("Affected area: left=%d, top=%d, width=%d, height=%d\n",
    options.Left, options.Top, options.Width, options.Height)
```
