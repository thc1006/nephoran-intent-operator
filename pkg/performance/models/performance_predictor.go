package models

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
)

// PerformancePredictor uses machine learning techniques for performance forecasting
type PerformancePredictor struct {
	trainingData    *mat.Dense
	performanceData *mat.Dense
	regressionModel *LinearRegressionModel
}

// LinearRegressionModel represents a simple linear regression model
type LinearRegressionModel struct {
	Coefficients []float64
	Intercept    float64
}

// NewPerformancePredictor initializes a new performance prediction model
func NewPerformancePredictor(features, performance [][]float64) *PerformancePredictor {
	trainingData := mat.NewDense(len(features), len(features[0]), flattenMatrix(features))
	performanceData := mat.NewDense(len(performance), len(performance[0]), flattenMatrix(performance))

	return &PerformancePredictor{
		trainingData:    trainingData,
		performanceData: performanceData,
	}
}

// Train performs machine learning model training
func (pp *PerformancePredictor) Train() error {
	// Multivariate linear regression
	nFeatures := pp.trainingData.RawMatrix().Cols
	nSamples := pp.trainingData.RawMatrix().Rows

	// Prepare problem for optimization
	problem := optimize.Problem{
		Func: func(x []float64) float64 {
			// Mean Squared Error loss function
			var mse float64
			for i := 0; i < nSamples; i++ {
				predicted := pp.predictSingle(x, pp.trainingData.RawRowView(i))
				actual := pp.performanceData.At(i, 0)
				diff := predicted - actual
				mse += diff * diff
			}
			return mse / float64(nSamples)
		},
		Grad: func(grad, x []float64) {
			// Gradient calculation for optimization
			// Simplified gradient descent implementation
			h := 1e-8
			for j := range x {
				xPlus := make([]float64, len(x))
				xMinus := make([]float64, len(x))
				copy(xPlus, x)
				copy(xMinus, x)

				xPlus[j] += h
				xMinus[j] -= h

				grad[j] = (optimize.Problem{Func: func(y []float64) float64 {
					return pp.evaluateLoss(y)
				}}.Func(xPlus) - pp.evaluateLoss(xMinus)) / (2 * h)
			}
		},
	}

	// Perform optimization
	result, err := optimize.Minimize(problem, make([]float64, nFeatures+1), nil, nil)
	if err != nil {
		return err
	}

	// Store model parameters
	pp.regressionModel = &LinearRegressionModel{
		Coefficients: result.X[:nFeatures],
		Intercept:    result.X[nFeatures],
	}

	return nil
}

// Predict generates performance predictions
func (pp *PerformancePredictor) Predict(features []float64) float64 {
	if pp.regressionModel == nil {
		panic("Model not trained")
	}
	return pp.predictSingle(pp.regressionModel.Coefficients, features) + pp.regressionModel.Intercept
}

// predictSingle is a helper for single prediction
func (pp *PerformancePredictor) predictSingle(coeffs, features []float64) float64 {
	prediction := 0.0
	for i, coeff := range coeffs {
		prediction += coeff * features[i]
	}
	return prediction
}

// evaluateLoss calculates model loss
func (pp *PerformancePredictor) evaluateLoss(params []float64) float64 {
	nFeatures := pp.trainingData.RawMatrix().Cols
	nSamples := pp.trainingData.RawMatrix().Rows

	var mse float64
	for i := 0; i < nSamples; i++ {
		predicted := pp.predictSingle(params[:nFeatures], pp.trainingData.RawRowView(i)) + params[nFeatures]
		actual := pp.performanceData.At(i, 0)
		diff := predicted - actual
		mse += diff * diff
	}
	return mse / float64(nSamples)
}

// Utility function to flatten 2D slice to 1D
func flattenMatrix(matrix [][]float64) []float64 {
	var flattened []float64
	for _, row := range matrix {
		flattened = append(flattened, row...)
	}
	return flattened
}

// PerformanceForecast represents a prediction result
type PerformanceForecast struct {
	PredictedValue     float64
	ConfidenceInterval struct {
		Lower float64
		Upper float64
	}
	Timestamp int64
}
