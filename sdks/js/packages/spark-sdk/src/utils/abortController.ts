let abortControllerImpl: typeof AbortController | null =
  globalThis.AbortController ?? null;

export const getAbortController = (): typeof AbortController => {
  if (!abortControllerImpl) {
    throw new Error(
      "AbortController implementation is not set. Please set it using setAbortController().",
    );
  }

  return abortControllerImpl;
};

export const setAbortController = (
  abortControllerImplParam: typeof AbortController,
): void => {
  abortControllerImpl = abortControllerImplParam;
  globalThis.AbortController = abortControllerImplParam;
};
