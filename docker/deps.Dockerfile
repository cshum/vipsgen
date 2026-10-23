FROM ubuntu:24.04

ARG ULTRAHDR_VERSION=v1.4.0

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
      meson ninja-build cmake git wget curl ca-certificates gpg sudo \
      libglib2.0-dev libexpat-dev librsvg2-dev libpng-dev \
      libjpeg-turbo8-dev libimagequant-dev libfftw3-dev \
      libpoppler-glib-dev libxml2-dev \
      libopenslide-dev libcfitsio-dev liborc-0.4-dev libpango1.0-dev \
      libtiff5-dev libgsf-1-dev giflib-tools libwebp-dev libheif-dev \
      libheif-plugin-x265 libheif-plugin-libde265 \
      libopenjp2-7-dev libcgif-dev \
      gobject-introspection libgirepository1.0-dev \
      libmagickwand-dev libmatio-dev libnifti2-dev \
      libjxl-dev libzip-dev libarchive-dev \
      libraw-dev \
 && rm -rf /var/lib/apt/lists/*

# Ubuntu's libnifti2-dev does not ship the pkg-config file libvips looks for.
RUN mkdir -p /usr/local/lib/pkgconfig \
 && printf '%s\n' \
      'prefix=/usr' \
      'exec_prefix=${prefix}' \
      'libdir=${prefix}/lib/x86_64-linux-gnu' \
      'includedir=${prefix}/include/nifti' \
      '' \
      'Name: libniftiio' \
      'Description: nifti library' \
      'Version: 3.0.1' \
      'Requires:' \
      'Cflags: -I${includedir}' \
      'Libs: -L${libdir} -lniftiio -lznz' \
      > /usr/local/lib/pkgconfig/niftiio.pc

# Build libultrahdr
RUN git clone --depth 1 --branch "${ULTRAHDR_VERSION}" https://github.com/google/libultrahdr.git /tmp/libultrahdr \
 && cmake -S /tmp/libultrahdr -B /tmp/libultrahdr/build \
      -DCMAKE_BUILD_TYPE=Release \
      -DUHDR_BUILD_EXAMPLES=OFF \
      -DUHDR_BUILD_TESTS=OFF \
      -DUHDR_BUILD_BENCHMARK=OFF \
 && cmake --build /tmp/libultrahdr/build -j"$(nproc)" \
 && cmake --install /tmp/libultrahdr/build \
 && ldconfig \
 && rm -rf /tmp/libultrahdr \
 && pkg-config --modversion libuhdr
