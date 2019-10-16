/*
 * Copyright 2014 The Kythe Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.google.devtools.kythe.extractors.java;

import com.google.common.collect.ImmutableSet;
import com.sun.tools.javac.code.Symbol.ClassSymbol;
import com.sun.tools.javac.code.Symbol.TypeSymbol;
import java.io.IOException;
import java.util.Arrays;
import java.util.Map;
import java.util.Set;
import java.util.logging.Level;
import java.util.logging.Logger;
import javax.annotation.processing.AbstractProcessor;
import javax.annotation.processing.RoundEnvironment;
import javax.annotation.processing.SupportedAnnotationTypes;
import javax.lang.model.SourceVersion;
import javax.lang.model.element.AnnotationMirror;
import javax.lang.model.element.AnnotationValue;
import javax.lang.model.element.Element;
import javax.lang.model.element.ExecutableElement;
import javax.lang.model.element.TypeElement;
import javax.lang.model.util.Elements;
import javax.tools.JavaFileObject;
import javax.tools.JavaFileObject.Kind;
import javax.tools.StandardLocation;

/**
 * This class is used to visit all the annotation used in Java files and record the usage of these
 * annotation as JavaFileObject. This processing will eliminate some of the platform errors caused
 * by javac not being able to find the class for some of the annotations in the analysis phase. The
 * recorded files will later be put in the datastore for analysis phase.
 */
@SupportedAnnotationTypes(value = {"*"})
public class ProcessAnnotation extends AbstractProcessor {

  private static final Logger logger = Logger.getLogger(ProcessAnnotation.class.getName());

  // ANDROID_BUILD: This line has been copied from ../../platform/shared/Metadata.java to avoid
  // compiling that file and all its dependencies when building javac_extractor for Android.
  private static final String ANNOTATION_COMMENT_PREFIX = "annotations:";

  private static final ImmutableSet<String> GENERATED_ANNOTATIONS =
      ImmutableSet.of("javax.annotation.Generated", "javax.annotation.processing.Generated");

  UsageAsInputReportingFileManager fileManager;

  public ProcessAnnotation(UsageAsInputReportingFileManager fileManager) {
    this.fileManager = fileManager;
  }

  @Override
  public boolean process(Set<? extends TypeElement> annotations, RoundEnvironment roundEnv) {
    for (TypeElement e : annotations) {
      TypeSymbol s = (TypeSymbol) e;
      try {
        UsageAsInputReportingJavaFileObject jfo =
            (UsageAsInputReportingJavaFileObject)
                fileManager.getJavaFileForInput(
                    StandardLocation.CLASS_OUTPUT, s.flatName().toString(), Kind.CLASS);
        if (jfo == null) {
          jfo =
              (UsageAsInputReportingJavaFileObject)
                  fileManager.getJavaFileForInput(
                      StandardLocation.CLASS_PATH, s.flatName().toString(), Kind.CLASS);
        }
        if (jfo == null) {
          jfo =
              (UsageAsInputReportingJavaFileObject)
                  fileManager.getJavaFileForInput(
                      StandardLocation.SOURCE_PATH, s.flatName().toString(), Kind.CLASS);
        }
        if (jfo != null) {
          jfo.markUsed();
        }
      } catch (IOException ex) {
        // We only log any IO exception here and do not cancel the whole processing because of an
        // exception in this stage.
        logger.log(Level.SEVERE, "Error in annotation processing", ex);
      }
    }

    Elements elementUtils = processingEnv.getElementUtils();
    for (String annotationName : GENERATED_ANNOTATIONS) {
      TypeElement annotationType = elementUtils.getTypeElement(annotationName);
      if (annotationType == null) {
        // javax.annotation.processing.Generated isn't available until Java 9
        continue;
      }
      for (Element ae : roundEnv.getElementsAnnotatedWith(annotationType)) {
        String comments = getGeneratedComments(elementUtils, ae, annotationType);
        if (comments == null || !comments.startsWith(ANNOTATION_COMMENT_PREFIX)) {
          continue;
        }
        String annotationFile = comments.substring(ANNOTATION_COMMENT_PREFIX.length());
        if (ae instanceof ClassSymbol) {
          ClassSymbol cs = (ClassSymbol) ae;
          try {
            String annotationPath = cs.sourcefile.toUri().resolve(annotationFile).getPath();
            for (JavaFileObject file :
                fileManager.getJavaFileForSources(Arrays.asList(annotationPath))) {
              ((UsageAsInputReportingJavaFileObject) file).markUsed();
            }
          } catch (IllegalArgumentException ex) {
            logger.log(Level.WARNING, "Bad annotationFile: " + annotationFile, ex);
          }
        }
      }
    }

    // We must return false so normal processors run after us.
    return false;
  }

  @Override
  public SourceVersion getSupportedSourceVersion() {
    return SourceVersion.latest();
  }

  private static String getGeneratedComments(
      Elements elementUtils, Element element, TypeElement annotationType) {
    AnnotationMirror mirror;
    for (AnnotationMirror annotationMirror : element.getAnnotationMirrors()) {
      Element el = annotationMirror.getAnnotationType().asElement();
      if (!annotationType.equals(el)) {
        continue;
      }
      Object value = null;
      for (Map.Entry<? extends ExecutableElement, ? extends AnnotationValue> entry :
          elementUtils.getElementValuesWithDefaults(annotationMirror).entrySet()) {
        if (entry.getKey().getSimpleName().contentEquals("comments")) {
          value = entry.getValue();
          break;
        }
      }
      if (value == null) {
        throw new IllegalArgumentException(
            String.format("@%s does not define an element comments()", element));
      }
      if (value instanceof String) {
        return (String) value;
      }
    }
    return null;
  }
}
